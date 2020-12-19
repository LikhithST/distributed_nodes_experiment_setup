package runner

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

// TickValue is the tick value
type TickValue struct {
	instant   time.Time
	reqNumber uint64
}

// Worker is used for doing a single stream of requests in parallel
type Worker struct {
	stub grpcdynamic.Stub
	mtd  *desc.MethodDescriptor

	config   *RunConfig
	workerID string
	active   bool
	stopCh   chan bool
	ticks    <-chan TickValue

	dataProvider *dataProvider
}

func (w *Worker) runWorker() error {
	var err error
	g := new(errgroup.Group)

	for {
		select {
		case <-w.stopCh:
			if w.config.async {
				return g.Wait()
			}

			return err
		case tv := <-w.ticks:
			if w.config.async {
				g.Go(func() error {
					return w.makeRequest(tv)
				})
			} else {
				rErr := w.makeRequest(tv)
				err = multierr.Append(err, rErr)
			}
		}
	}
}

// Stop stops the worker. It has to be started with Run() again.
func (w *Worker) Stop() {
	if !w.active {
		return
	}

	w.active = false
	w.stopCh <- true
}

func (w *Worker) makeRequest(tv TickValue) error {
	reqNum := int64(tv.reqNumber)

	ctd := newCallData(w.mtd, w.config.funcs, w.workerID, reqNum)

	inputs, err := w.dataProvider.getDataForCall(ctd)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return fmt.Errorf("no data provided for request")
	}

	reqMD, err := w.dataProvider.getMetadataForCall(ctd)
	if err != nil {
		return err
	}

	if w.config.enableCompression {
		reqMD.Append("grpc-accept-encoding", gzip.Name)
	}

	ctx := context.Background()
	var cancel context.CancelFunc

	if w.config.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, w.config.timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// include the metadata
	if reqMD != nil {
		ctx = metadata.NewOutgoingContext(ctx, *reqMD)
	}

	var callType string
	if w.config.hasLog {
		callType = "unary"
		if w.mtd.IsClientStreaming() && w.mtd.IsServerStreaming() {
			callType = "bidi"
		} else if w.mtd.IsServerStreaming() {
			callType = "server-streaming"
		} else if w.mtd.IsClientStreaming() {
			callType = "client-streaming"
		}

		w.config.log.Debugw("Making request", "workerID", w.workerID,
			"call type", callType, "call", w.mtd.GetFullyQualifiedName(),
			"input", inputs, "metadata", reqMD)
	}

	unaryInput := inputs[0]

	// RPC errors are handled via stats handler
	if w.mtd.IsClientStreaming() && w.mtd.IsServerStreaming() {
		_ = w.makeBidiRequest(&ctx, inputs)
	} else if w.mtd.IsClientStreaming() {
		_ = w.makeClientStreamingRequest(&ctx, inputs)
	} else if w.mtd.IsServerStreaming() {
		_ = w.makeServerStreamingRequest(&ctx, unaryInput)
	} else {
		_ = w.makeUnaryRequest(&ctx, reqMD, unaryInput)
	}

	return err
}

func (w *Worker) makeUnaryRequest(ctx *context.Context, reqMD *metadata.MD, input *dynamic.Message) error {
	var res proto.Message
	var resErr error
	var callOptions = []grpc.CallOption{}
	if w.config.enableCompression {
		callOptions = append(callOptions, grpc.UseCompressor(gzip.Name))
	}

	res, resErr = w.stub.InvokeRpc(*ctx, w.mtd, input, callOptions...)

	if w.config.hasLog {
		w.config.log.Debugw("Received response", "workerID", w.workerID, "call type", "unary",
			"call", w.mtd.GetFullyQualifiedName(),
			"input", input, "metadata", reqMD,
			"response", res, "error", resErr)
	}

	return resErr
}

func (w *Worker) makeClientStreamingRequest(ctx *context.Context, input []*dynamic.Message) error {
	var str *grpcdynamic.ClientStream
	var err error
	var callOptions = []grpc.CallOption{}
	if w.config.enableCompression {
		callOptions = append(callOptions, grpc.UseCompressor(gzip.Name))
	}
	str, err = w.stub.InvokeRpcClientStream(*ctx, w.mtd, callOptions...)

	if err != nil && w.config.hasLog {
		w.config.log.Errorw("Invoke Client Streaming RPC call error: "+err.Error(), "workerID", w.workerID,
			"call type", "client-streaming",
			"call", w.mtd.GetFullyQualifiedName(), "error", err)
	}

	counter := 0

	closeStream := func() {
		res, closeErr := str.CloseAndReceive()

		if w.config.hasLog {
			w.config.log.Debugw("Close and receive", "workerID", w.workerID, "call type", "client-streaming",
				"call", w.mtd.GetFullyQualifiedName(),
				"response", res, "error", closeErr)
		}
	}

	performSend := func() bool {
		inputLen := len(input)
		if input == nil || inputLen == 0 {
			return true
		}

		if counter == inputLen {
			return true
		}

		payload := input[counter]

		err = str.SendMsg(payload)

		if w.config.hasLog {
			w.config.log.Debugw("Send message", "workerID", w.workerID, "call type", "client-streaming",
				"call", w.mtd.GetFullyQualifiedName(),
				"payload", payload, "error", err)
		}

		if err == io.EOF {
			return true
		}

		counter++

		return false
	}

	cancel := make(chan struct{}, 1)
	if w.config.streamClose > 0 {
		go func() {
			sct := time.NewTimer(w.config.streamClose)
			<-sct.C
			cancel <- struct{}{}
		}()
	}

	done := false
	for err == nil && !done {

		if end := performSend(); end {
			closeStream()
			break
		}

		if w.config.streamInterval > 0 {
			wait := time.NewTimer(w.config.streamInterval)
			select {
			case <-wait.C:
				break
			case <-cancel:
				closeStream()
				done = true
			}
		} else if w.config.streamClose > 0 && len(cancel) > 0 {
			<-cancel
			closeStream()
			done = true
		}

		if done {
			break
		}
	}

	close(cancel)

	return nil
}

func (w *Worker) makeServerStreamingRequest(ctx *context.Context, input *dynamic.Message) error {
	var callOptions = []grpc.CallOption{}
	if w.config.enableCompression {
		callOptions = append(callOptions, grpc.UseCompressor(gzip.Name))
	}
	str, err := w.stub.InvokeRpcServerStream(*ctx, w.mtd, input, callOptions...)

	if err != nil && w.config.hasLog {
		w.config.log.Errorw("Invoke Server Streaming RPC call error: "+err.Error(), "workerID", w.workerID,
			"call type", "server-streaming",
			"call", w.mtd.GetFullyQualifiedName(),
			"input", input, "error", err)
	}

	for err == nil {
		res, err := str.RecvMsg()

		if w.config.hasLog {
			w.config.log.Debugw("Receive message", "workerID", w.workerID, "call type", "server-streaming",
				"call", w.mtd.GetFullyQualifiedName(),
				"response", res, "error", err)
		}

		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
	}

	return err
}

func (w *Worker) makeBidiRequest(ctx *context.Context, input []*dynamic.Message) error {
	var str *grpcdynamic.BidiStream
	var err error
	var callOptions = []grpc.CallOption{}
	if w.config.enableCompression {
		callOptions = append(callOptions, grpc.UseCompressor(gzip.Name))
	}
	str, err = w.stub.InvokeRpcBidiStream(*ctx, w.mtd, callOptions...)

	if err != nil {
		if w.config.hasLog {
			w.config.log.Errorw("Invoke Bidi RPC call error: "+err.Error(),
				"workerID", w.workerID, "call type", "bidi",
				"call", w.mtd.GetFullyQualifiedName(), "error", err)
		}

		return err
	}

	counter := 0

	inputLen := len(input)

	recvDone := make(chan bool)

	if input == nil || inputLen == 0 {
		closeErr := str.CloseSend()

		if w.config.hasLog {
			w.config.log.Debugw("Close send", "workerID", w.workerID, "call type", "bidi",
				"call", w.mtd.GetFullyQualifiedName(), "error", closeErr)
		}

		return nil
	}

	go func() {
		for {
			res, err := str.RecvMsg()

			if w.config.hasLog {
				w.config.log.Debugw("Receive message", "workerID", w.workerID, "call type", "bidi",
					"call", w.mtd.GetFullyQualifiedName(),
					"response", res, "error", err)
			}

			if err != nil {
				close(recvDone)
				break
			}
		}
	}()

	closeStream := func() {
		closeErr := str.CloseSend()

		if w.config.hasLog {
			w.config.log.Debugw("Close send", "workerID", w.workerID, "call type", "bidi",
				"call", w.mtd.GetFullyQualifiedName(), "error", closeErr)
		}
	}

	var finished uint32
	if w.config.streamClose > 0 {
		go func() {
			sct := time.NewTimer(w.config.streamClose)
			<-sct.C
			atomic.AddUint32(&finished, 1)
		}()
	}

	for err == nil {

		if counter == inputLen {
			closeStream()
			break
		}

		payload := input[counter]

		// we need to check before and after stream interval
		toClose := atomic.LoadUint32(&finished)
		if toClose > 0 {
			closeStream()
			break
		}

		var wait <-chan time.Time
		if w.config.streamInterval > 0 {
			wait = time.Tick(w.config.streamInterval)
			<-wait
		}

		toClose = atomic.LoadUint32(&finished)
		if toClose > 0 {
			closeStream()
			break
		}

		err = str.SendMsg(payload)
		counter++

		if w.config.hasLog {
			w.config.log.Debugw("Send message", "workerID", w.workerID, "call type", "bidi",
				"call", w.mtd.GetFullyQualifiedName(),
				"payload", payload, "error", err)
		}
	}

	if err == nil {
		<-recvDone
	}

	return nil
}
