/*
 *
 * Copyright 2014 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package grpc

import (
	"errors"
	"math"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/transport"
	"fmt"
)

var (
	// ErrClientConnClosing indicates that the operation is illegal because
	// the ClientConn is closing.
	ErrClientConnClosing = errors.New("grpc: the client connection is closing")
	// ErrClientConnTimeout indicates that the ClientConn cannot establish the
	// underlying connections within the specified timeout.
	// DEPRECATED: Please use context.DeadlineExceeded instead.
	ErrClientConnTimeout = errors.New("grpc: timed out when dialing")

	// errNoTransportSecurity indicates that there is no transport security
	// being set for ClientConn. Users should either set one or explicitly
	// call WithInsecure DialOption to disable security.
	errNoTransportSecurity = errors.New("grpc: no transport security set (use grpc.WithInsecure() explicitly or set credentials)")
	// errTransportCredentialsMissing indicates that users want to transmit security
	// information (e.g., oauth2 token) which requires secure connection on an insecure
	// connection.
	errTransportCredentialsMissing = errors.New("grpc: the credentials require transport level security (use grpc.WithTransportCredentials() to set)")
	// errCredentialsConflict indicates that grpc.WithTransportCredentials()
	// and grpc.WithInsecure() are both called for a connection.
	errCredentialsConflict = errors.New("grpc: transport credentials are set for an insecure connection (grpc.WithTransportCredentials() and grpc.WithInsecure() are both called)")
	// errNetworkIO indicates that the connection is down due to some network I/O error.
	errNetworkIO = errors.New("grpc: failed with network I/O error")
	// errConnDrain indicates that the connection starts to be drained and does not accept any new RPCs.
	errConnDrain = errors.New("grpc: the connection is drained")
	// errConnClosing indicates that the connection is closing.
	errConnClosing = errors.New("grpc: the connection is closing")
	// errConnUnavailable indicates that the connection is unavailable.
	errConnUnavailable = errors.New("grpc: the connection is unavailable")
	// errBalancerClosed indicates that the balancer is closed.
	errBalancerClosed = errors.New("grpc: balancer is closed")
	// minimum time to give a connection to complete
	minConnectTimeout = 20 * time.Second
)

// dialOptions configure a Dial call. dialOptions are set by the DialOption
// values passed to Dial.
type dialOptions struct {
	unaryInt    UnaryClientInterceptor
	streamInt   StreamClientInterceptor
	codec       Codec
	cp          Compressor
	dc          Decompressor
	bs          backoffStrategy
	block       bool
	insecure    bool
	timeout     time.Duration
	scChan      <-chan ServiceConfig
	copts       transport.ConnectOptions
	callOptions []CallOption
	// This is to support v1 balancer.
	balancerBuilder balancer.Builder
}

const (
	defaultClientMaxReceiveMessageSize = 1024 * 1024 * 4
	defaultClientMaxSendMessageSize    = math.MaxInt32
)

// DialOption configures how we set up the connection.
type DialOption func(*dialOptions)

// WithInitialWindowSize returns a DialOption which sets the value for initial window size on a stream.
// The lower bound for window size is 64K and any value smaller than that will be ignored.
func WithInitialWindowSize(s int32) DialOption {
	return func(o *dialOptions) {
		o.copts.InitialWindowSize = s
	}
}

// WithInitialConnWindowSize returns a DialOption which sets the value for initial window size on a connection.
// The lower bound for window size is 64K and any value smaller than that will be ignored.
func WithInitialConnWindowSize(s int32) DialOption {
	return func(o *dialOptions) {
		o.copts.InitialConnWindowSize = s
	}
}

// WithMaxMsgSize returns a DialOption which sets the maximum message size the client can receive. Deprecated: use WithDefaultCallOptions(MaxCallRecvMsgSize(s)) instead.
func WithMaxMsgSize(s int) DialOption {
	return WithDefaultCallOptions(MaxCallRecvMsgSize(s))
}

// WithDefaultCallOptions returns a DialOption which sets the default CallOptions for calls over the connection.
func WithDefaultCallOptions(cos ...CallOption) DialOption {
	return func(o *dialOptions) {
		o.callOptions = append(o.callOptions, cos...)
	}
}

// WithCodec returns a DialOption which sets a codec for message marshaling and unmarshaling.
func WithCodec(c Codec) DialOption {
	return func(o *dialOptions) {
		o.codec = c
	}
}

// WithCompressor returns a DialOption which sets a CompressorGenerator for generating message
// compressor.
func WithCompressor(cp Compressor) DialOption {
	return func(o *dialOptions) {
		o.cp = cp
	}
}

// WithDecompressor returns a DialOption which sets a DecompressorGenerator for generating
// message decompressor.
func WithDecompressor(dc Decompressor) DialOption {
	return func(o *dialOptions) {
		o.dc = dc
	}
}

// WithBalancer returns a DialOption which sets a load balancer with the v1 API.
// Name resolver will be ignored if this DialOption is specified.
// Deprecated: use the new balancer APIs in balancer package instead.
func WithBalancer(b Balancer) DialOption {
	return func(o *dialOptions) {
		o.balancerBuilder = &balancerWrapperBuilder{
			b: b,
		}
	}
}

// WithServiceConfig returns a DialOption which has a channel to read the service configuration.
func WithServiceConfig(c <-chan ServiceConfig) DialOption {
	return func(o *dialOptions) {
		o.scChan = c
	}
}

// WithBackoffMaxDelay configures the dialer to use the provided maximum delay
// when backing off after failed connection attempts.
func WithBackoffMaxDelay(md time.Duration) DialOption {
	return WithBackoffConfig(BackoffConfig{MaxDelay: md})
}

// WithBackoffConfig configures the dialer to use the provided backoff
// parameters after connection failures.
//
// Use WithBackoffMaxDelay until more parameters on BackoffConfig are opened up
// for use.
func WithBackoffConfig(b BackoffConfig) DialOption {
	// Set defaults to ensure that provided BackoffConfig is valid and
	// unexported fields get default values.
	setDefaults(&b)
	return withBackoff(b)
}

// withBackoff sets the backoff strategy used for retries after a
// failed connection attempt.
//
// This can be exported if arbitrary backoff strategies are allowed by gRPC.
func withBackoff(bs backoffStrategy) DialOption {
	return func(o *dialOptions) {
		o.bs = bs
	}
}

// WithBlock returns a DialOption which makes caller of Dial blocks until the underlying
// connection is up. Without this, Dial returns immediately and connecting the server
// happens in background.
func WithBlock() DialOption {
	return func(o *dialOptions) {
		o.block = true
	}
}

// WithInsecure returns a DialOption which disables transport security for this ClientConn.
// Note that transport security is required unless WithInsecure is set.
func WithInsecure() DialOption {
	return func(o *dialOptions) {
		o.insecure = true
	}
}

// WithTransportCredentials returns a DialOption which configures a
// connection level security credentials (e.g., TLS/SSL).
func WithTransportCredentials(creds credentials.TransportCredentials) DialOption {
	return func(o *dialOptions) {
		o.copts.TransportCredentials = creds
	}
}

// WithPerRPCCredentials returns a DialOption which sets
// credentials and places auth state on each outbound RPC.
func WithPerRPCCredentials(creds credentials.PerRPCCredentials) DialOption {
	return func(o *dialOptions) {
		o.copts.PerRPCCredentials = append(o.copts.PerRPCCredentials, creds)
	}
}

// WithTimeout returns a DialOption that configures a timeout for dialing a ClientConn
// initially. This is valid if and only if WithBlock() is present.
// Deprecated: use DialContext and context.WithTimeout instead.
func WithTimeout(d time.Duration) DialOption {
	return func(o *dialOptions) {
		o.timeout = d
	}
}

// WithDialer returns a DialOption that specifies a function to use for dialing network addresses.
// If FailOnNonTempDialError() is set to true, and an error is returned by f, gRPC checks the error's
// Temporary() method to decide if it should try to reconnect to the network address.
func WithDialer(f func(string, time.Duration) (net.Conn, error)) DialOption {
	return func(o *dialOptions) {
		o.copts.Dialer = func(ctx context.Context, addr string) (net.Conn, error) {
			if deadline, ok := ctx.Deadline(); ok {
				return f(addr, deadline.Sub(time.Now()))
			}
			return f(addr, 0)
		}
	}
}

// WithStatsHandler returns a DialOption that specifies the stats handler
// for all the RPCs and underlying network connections in this ClientConn.
func WithStatsHandler(h stats.Handler) DialOption {
	return func(o *dialOptions) {
		o.copts.StatsHandler = h
	}
}

// FailOnNonTempDialError returns a DialOption that specifies if gRPC fails on non-temporary dial errors.
// If f is true, and dialer returns a non-temporary error, gRPC will fail the connection to the network
// address and won't try to reconnect.
// The default value of FailOnNonTempDialError is false.
// This is an EXPERIMENTAL API.
func FailOnNonTempDialError(f bool) DialOption {
	return func(o *dialOptions) {
		o.copts.FailOnNonTempDialError = f
	}
}

// WithUserAgent returns a DialOption that specifies a user agent string for all the RPCs.
func WithUserAgent(s string) DialOption {
	return func(o *dialOptions) {
		o.copts.UserAgent = s
	}
}

// WithKeepaliveParams returns a DialOption that specifies keepalive paramaters for the client transport.
func WithKeepaliveParams(kp keepalive.ClientParameters) DialOption {
	return func(o *dialOptions) {
		o.copts.KeepaliveParams = kp
	}
}

// WithUnaryInterceptor returns a DialOption that specifies the interceptor for unary RPCs.
func WithUnaryInterceptor(f UnaryClientInterceptor) DialOption {
	return func(o *dialOptions) {
		o.unaryInt = f
	}
}

// WithStreamInterceptor returns a DialOption that specifies the interceptor for streaming RPCs.
func WithStreamInterceptor(f StreamClientInterceptor) DialOption {
	return func(o *dialOptions) {
		o.streamInt = f
	}
}

// WithAuthority returns a DialOption that specifies the value to be used as
// the :authority pseudo-header. This value only works with WithInsecure and
// has no effect if TransportCredentials are present.
func WithAuthority(a string) DialOption {
	return func(o *dialOptions) {
		o.copts.Authority = a
	}
}

// Dial creates a client connection to the given target.
func Dial(target string, opts ...DialOption) (*ClientConn, error) {
	return DialContext(context.Background(), target, opts...)
}

// DialContext creates a client connection to the given target. ctx can be used to
// cancel or expire the pending connection. Once this function returns, the
// cancellation and expiration of ctx will be noop. Users should call ClientConn.Close
// to terminate all the pending operations after this function returns.
func DialContext(ctx context.Context, target string, opts ...DialOption) (conn *ClientConn, err error) {
	cc := &ClientConn{
		target: target,
		csMgr:  &connectivityStateManager{},
		conns:  make(map[*addrConn]struct{}),
	}
	cc.csEvltr = &connectivityStateEvaluator{csMgr: cc.csMgr}
	cc.ctx, cc.cancel = context.WithCancel(context.Background())

	for _, opt := range opts {
		opt(&cc.dopts)
	}
	cc.mkp = cc.dopts.copts.KeepaliveParams

	if cc.dopts.copts.Dialer == nil {
		cc.dopts.copts.Dialer = newProxyDialer(
			func(ctx context.Context, addr string) (net.Conn, error) {
				return dialContext(ctx, "tcp", addr)
			},
		)
	}

	if cc.dopts.copts.UserAgent != "" {
		cc.dopts.copts.UserAgent += " " + grpcUA
	} else {
		cc.dopts.copts.UserAgent = grpcUA
	}

	if cc.dopts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cc.dopts.timeout)
		defer cancel()
	}

	defer func() {
		select {
		case <-ctx.Done():
			conn, err = nil, ctx.Err()
		default:
		}

		if err != nil {
			cc.Close()
		}
	}()

	scSet := false
	if cc.dopts.scChan != nil {
		// Try to get an initial service config.
		select {
		case sc, ok := <-cc.dopts.scChan:
			if ok {
				cc.sc = sc
				scSet = true
			}
		default:
		}
	}
	// Set defaults.
	if cc.dopts.codec == nil {
		cc.dopts.codec = protoCodec{}
	}
	if cc.dopts.bs == nil {
		cc.dopts.bs = DefaultBackoffConfig
	}
	creds := cc.dopts.copts.TransportCredentials
	if creds != nil && creds.Info().ServerName != "" {
		cc.authority = creds.Info().ServerName
	} else if cc.dopts.insecure && cc.dopts.copts.Authority != "" {
		cc.authority = cc.dopts.copts.Authority
	} else {
		cc.authority = target
	}

	// TODO(bar) parse scheme and start resolver.
	if cc.dopts.balancerBuilder != nil {
		var credsClone credentials.TransportCredentials
		if creds != nil {
			credsClone = creds.Clone()
		}
		buildOpts := balancer.BuildOptions{
			DialCreds: credsClone,
			Dialer:    cc.dopts.copts.Dialer,
		}
		// Build should not take long time. So it's ok to not have a goroutine for it.
		// TODO(bar) init balancer after first resolver result to support service config balancer.
		cc.balancer = cc.dopts.balancerBuilder.Build(&ccBalancerWrapper{cc: cc}, buildOpts)
	} else {
		waitC := make(chan error, 1)
		go func() {
			defer close(waitC)
			// No balancer, or no resolver within the balancer.  Connect directly.
			ac, err := cc.newAddrConn([]resolver.Address{{Addr: target}})
			if err != nil {
				waitC <- err
				return
			}
			if err := ac.connect(cc.dopts.block); err != nil {
				waitC <- err
				return
			}
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-waitC:
			if err != nil {
				return nil, err
			}
		}
	}
	if cc.dopts.scChan != nil && !scSet {
		// Blocking wait for the initial service config.
		select {
		case sc, ok := <-cc.dopts.scChan:
			if ok {
				cc.sc = sc
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if cc.dopts.scChan != nil {
		go cc.scWatcher()
	}

	if cc.balancer != nil {
		// Unblock balancer initialization with a fake resolver update.
		// The balancer wrapper will not read the addresses, so an empty list works.
		// TODO(bar) remove this after the real resolver is started.
		cc.balancer.HandleResolvedAddrs([]resolver.Address{}, nil)
	}

	// A blocking dial blocks until the clientConn is ready.
	if cc.dopts.block {
		for {
			s := cc.GetState()
			if s == connectivity.Ready {
				break
			}
			if !cc.WaitForStateChange(ctx, s) {
				// ctx got timeout or canceled.
				return nil, ctx.Err()
			}
		}
	}

	return cc, nil
}

// connectivityStateEvaluator gets updated by addrConns when their
// states transition, based on which it evaluates the state of
// ClientConn.
// Note: This code will eventually sit in the balancer in the new design.
type connectivityStateEvaluator struct {
	csMgr               *connectivityStateManager
	mu                  sync.Mutex
	numReady            uint64 // Number of addrConns in ready state.
	numConnecting       uint64 // Number of addrConns in connecting state.
	numTransientFailure uint64 // Number of addrConns in transientFailure.
}

// recordTransition records state change happening in every addrConn and based on
// that it evaluates what state the ClientConn is in.
// It can only transition between connectivity.Ready, connectivity.Connecting and connectivity.TransientFailure. Other states,
// Idle and connectivity.Shutdown are transitioned into by ClientConn; in the begining of the connection
// before any addrConn is created ClientConn is in idle state. In the end when ClientConn
// closes it is in connectivity.Shutdown state.
// TODO Note that in later releases, a ClientConn with no activity will be put into an Idle state.
func (cse *connectivityStateEvaluator) recordTransition(oldState, newState connectivity.State) {
	cse.mu.Lock()
	defer cse.mu.Unlock()

	// Update counters.
	for idx, state := range []connectivity.State{oldState, newState} {
		updateVal := 2*uint64(idx) - 1 // -1 for oldState and +1 for new.
		switch state {
		case connectivity.Ready:
			cse.numReady += updateVal
		case connectivity.Connecting:
			cse.numConnecting += updateVal
		case connectivity.TransientFailure:
			cse.numTransientFailure += updateVal
		}
	}

	// Evaluate.
	if cse.numReady > 0 {
		cse.csMgr.updateState(connectivity.Ready)
		return
	}
	if cse.numConnecting > 0 {
		cse.csMgr.updateState(connectivity.Connecting)
		return
	}
	cse.csMgr.updateState(connectivity.TransientFailure)
}

// connectivityStateManager keeps the connectivity.State of ClientConn.
// This struct will eventually be exported so the balancers can access it.
type connectivityStateManager struct {
	mu         sync.Mutex
	state      connectivity.State
	notifyChan chan struct{}
}

// updateState updates the connectivity.State of ClientConn.
// If there's a change it notifies goroutines waiting on state change to
// happen.
func (csm *connectivityStateManager) updateState(state connectivity.State) {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	if csm.state == connectivity.Shutdown {
		return
	}
	if csm.state == state {
		return
	}
	csm.state = state
	if csm.notifyChan != nil {
		// There are other goroutines waiting on this channel.
		close(csm.notifyChan)
		csm.notifyChan = nil
	}
}

func (csm *connectivityStateManager) getState() connectivity.State {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	return csm.state
}

func (csm *connectivityStateManager) getNotifyChan() <-chan struct{} {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	if csm.notifyChan == nil {
		csm.notifyChan = make(chan struct{})
	}
	return csm.notifyChan
}

// ClientConn represents a client connection to an RPC server.
type ClientConn struct {
	ctx    context.Context
	cancel context.CancelFunc

	target    string
	authority string
	dopts     dialOptions
	csMgr     *connectivityStateManager
	csEvltr   *connectivityStateEvaluator // This will eventually be part of balancer.

	balancer balancer.Balancer

	// TODO(bar) move the mutex and picker into a struct that does blocking pick().
	pmu    sync.Mutex
	picker balancer.Picker

	mu    sync.RWMutex
	sc    ServiceConfig
	conns map[*addrConn]struct{}
	// Keepalive parameter can be updated if a GoAway is received.
	mkp keepalive.ClientParameters
}

// WaitForStateChange waits until the connectivity.State of ClientConn changes from sourceState or
// ctx expires. A true value is returned in former case and false in latter.
// This is an EXPERIMENTAL API.
func (cc *ClientConn) WaitForStateChange(ctx context.Context, sourceState connectivity.State) bool {
	ch := cc.csMgr.getNotifyChan()
	if cc.csMgr.getState() != sourceState {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-ch:
		return true
	}
}

// GetState returns the connectivity.State of ClientConn.
// This is an EXPERIMENTAL API.
func (cc *ClientConn) GetState() connectivity.State {
	return cc.csMgr.getState()
}

func (cc *ClientConn) scWatcher() {
	for {
		select {
		case sc, ok := <-cc.dopts.scChan:
			if !ok {
				return
			}
			cc.mu.Lock()
			// TODO: load balance policy runtime change is ignored.
			// We may revist this decision in the future.
			cc.sc = sc
			cc.mu.Unlock()
		case <-cc.ctx.Done():
			return
		}
	}
}

// newAddrConn creates an addrConn for addrs and adds it to cc.conns.
func (cc *ClientConn) newAddrConn(addrs []resolver.Address) (*addrConn, error) {
	ac := &addrConn{
		cc:    cc,
		addrs: addrs,
		dopts: cc.dopts,
	}
	ac.ctx, ac.cancel = context.WithCancel(cc.ctx)
	ac.csEvltr = cc.csEvltr
	// Track ac in cc. This needs to be done before any getTransport(...) is called.
	cc.mu.Lock()
	if cc.conns == nil {
		cc.mu.Unlock()
		return nil, ErrClientConnClosing
	}
	cc.conns[ac] = struct{}{}
	cc.mu.Unlock()
	return ac, nil
}

// removeAddrConn removes the addrConn in the subConn from clientConn.
// It also tears down the ac with the given error.
func (cc *ClientConn) removeAddrConn(ac *addrConn, err error) {
	cc.mu.Lock()
	if cc.conns == nil {
		cc.mu.Unlock()
		return
	}
	delete(cc.conns, ac)
	cc.mu.Unlock()
	ac.tearDown(err)
}

// connect starts to creating transport and also starts the transport monitor
// goroutine for this ac.
// TODO(bar) Move this to the addrConn section.
// This was part of resetAddrConn, keep it here to make the diff look clean.
func (ac *addrConn) connect(block bool) error {
	ac.mu.Lock()
	if ac.state == connectivity.Shutdown {
		fmt.Println("in clientconn connect 667 closing")
		ac.mu.Unlock()
		return errConnClosing
	}
	ac.mu.Unlock()

	if EnableTracing {
		ac.events = trace.NewEventLog("grpc.ClientConn", ac.addrs[0].Addr)
	}
	if !ac.dopts.insecure {
		if ac.dopts.copts.TransportCredentials == nil {
			return errNoTransportSecurity
		}
	} else {
		if ac.dopts.copts.TransportCredentials != nil {
			return errCredentialsConflict
		}
		for _, cd := range ac.dopts.copts.PerRPCCredentials {
			if cd.RequireTransportSecurity() {
				return errTransportCredentialsMissing
			}
		}
	}

	if block {
		if err := ac.resetTransport(false); err != nil {
			if err != errConnClosing {
				ac.tearDown(err)
			}
			if e, ok := err.(transport.ConnectionError); ok && !e.Temporary() {
				return e.Origin()
			}
			return err
		}
		// Start to monitor the error status of transport.
		go ac.transportMonitor()
	} else {
		// Start a goroutine connecting to the server asynchronously.
		go func() {
			if err := ac.resetTransport(false); err != nil {
				grpclog.Warningf("Failed to dial %s: %v; please retry.", ac.addrs[0].Addr, err)
				if err != errConnClosing {
					// Keep this ac in cc.conns, to get the reason it's torn down.
					ac.tearDown(err)
				}
				return
			}
			ac.transportMonitor()
		}()
	}
	return nil
}

// tryUpdateAddrs tries to update ac.addrs with the new addresses list.
//
// It checks whether current connected address of ac is in the new addrs list.
//  - If true, it updates ac.addrs and returns true. The ac will keep using
//    the existing connection.
//  - If false, it does nothing and returns false.
func (ac *addrConn) tryUpdateAddrs(addrs []resolver.Address) bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	grpclog.Infof("addrConn: tryUpdateAddrs curAddr: %v, addrs: %v", ac.curAddr, addrs)
	if ac.state == connectivity.Shutdown {
		ac.addrs = addrs
		return true
	}

	var curAddrFound bool
	for _, a := range addrs {
		if reflect.DeepEqual(ac.curAddr, a) {
			curAddrFound = true
			break
		}
	}
	grpclog.Infof("addrConn: tryUpdateAddrs curAddrFound: %v", curAddrFound)
	if curAddrFound {
		ac.addrs = addrs
	}

	return curAddrFound
}

// GetMethodConfig gets the method config of the input method.
// If there's an exact match for input method (i.e. /service/method), we return
// the corresponding MethodConfig.
// If there isn't an exact match for the input method, we look for the default config
// under the service (i.e /service/). If there is a default MethodConfig for
// the serivce, we return it.
// Otherwise, we return an empty MethodConfig.
func (cc *ClientConn) GetMethodConfig(method string) MethodConfig {
	// TODO: Avoid the locking here.
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	m, ok := cc.sc.Methods[method]
	if !ok {
		i := strings.LastIndex(method, "/")
		m, _ = cc.sc.Methods[method[:i+1]]
	}
	return m
}

func (cc *ClientConn) getTransport(ctx context.Context, opts BalancerGetOptions) (transport.ClientTransport, func(balancer.DoneInfo), error) {
	var (
		ac  *addrConn
		put func(balancer.DoneInfo)
	)
	if cc.balancer == nil {
		// If balancer is nil, there should be only one addrConn available.
		cc.mu.RLock()
		if cc.conns == nil {
			cc.mu.RUnlock()
			return nil, nil, toRPCErr(ErrClientConnClosing)
		}
		for ac = range cc.conns {
			// Break after the first iteration to get the first addrConn.
			break
		}
		cc.mu.RUnlock()
	} else {
	    fmt.Println("in clientconn getTransport 787 clientconn.balancer is not nil")
		cc.pmu.Lock()
		// TODO(bar) call pick on struct blockPicker instead of the real picker.
		p := cc.picker
		cc.pmu.Unlock()

		var (
			err error
			sc  balancer.SubConn
		)
		sc, put, err = p.Pick(ctx, balancer.PickOptions{})
		if err != nil {
            fmt.Println("in clientconn getTransport 799 p.Pick err:", err)
            return nil, nil, toRPCErr(err)
		}
		if acbw, ok := sc.(*acBalancerWrapper); ok {
		    fmt.Println("in clientconn getTransport 803 before getAddrConn")
			ac = acbw.getAddrConn()
			fmt.Println("in clientconn getTransport 805, ac:", ac)
		} else if put != nil {
			updateRPCInfoInContext(ctx, rpcInfo{bytesSent: false, bytesReceived: false})
			put(balancer.DoneInfo{Err: errors.New("SubConn returned by pick cannot be recognized")})
		}
	}
	if ac == nil {
	    fmt.Println("in clientconn getTransport 811 ac == nil err:", errConnClosing)
		return nil, nil, errConnClosing
	}
	t, err := ac.wait(ctx, cc.balancer != nil, !opts.BlockingWait)
	if err != nil {
		if put != nil {
			updateRPCInfoInContext(ctx, rpcInfo{bytesSent: false, bytesReceived: false})
			put(balancer.DoneInfo{Err: err})
		}
		return nil, nil, err
	}
	return t, put, nil
}

// Close tears down the ClientConn and all underlying connections.
func (cc *ClientConn) Close() error {
	cc.cancel()

	cc.mu.Lock()
	if cc.conns == nil {
		cc.mu.Unlock()
		return ErrClientConnClosing
	}
	conns := cc.conns
	cc.conns = nil
	cc.csMgr.updateState(connectivity.Shutdown)
	cc.mu.Unlock()
	if cc.balancer != nil {
		cc.balancer.Close()
	}
	for ac := range conns {
		ac.tearDown(ErrClientConnClosing)
	}
	return nil
}

// addrConn is a network connection to a given address.
type addrConn struct {
	ctx    context.Context
	cancel context.CancelFunc

	cc      *ClientConn
	curAddr resolver.Address
	addrs   []resolver.Address
	dopts   dialOptions
	events  trace.EventLog
	acbw    balancer.SubConn

	csEvltr *connectivityStateEvaluator

	mu    sync.Mutex
	state connectivity.State
	// ready is closed and becomes nil when a new transport is up or failed
	// due to timeout.
	ready     chan struct{}
	transport transport.ClientTransport

	// The reason this addrConn is torn down.
	tearDownErr error
}

// adjustParams updates parameters used to create transports upon
// receiving a GoAway.
func (ac *addrConn) adjustParams(r transport.GoAwayReason) {
	switch r {
	case transport.TooManyPings:
		v := 2 * ac.dopts.copts.KeepaliveParams.Time
		ac.cc.mu.Lock()
		if v > ac.cc.mkp.Time {
			ac.cc.mkp.Time = v
		}
		ac.cc.mu.Unlock()
	}
}

// printf records an event in ac's event log, unless ac has been closed.
// REQUIRES ac.mu is held.
func (ac *addrConn) printf(format string, a ...interface{}) {
	if ac.events != nil {
		ac.events.Printf(format, a...)
	}
}

// errorf records an error in ac's event log, unless ac has been closed.
// REQUIRES ac.mu is held.
func (ac *addrConn) errorf(format string, a ...interface{}) {
	if ac.events != nil {
		ac.events.Errorf(format, a...)
	}
}

// resetTransport recreates a transport to the address for ac.
// For the old transport:
// - if drain is true, it will be gracefully closed.
// - otherwise, it will be closed.
// TODO(bar) make sure all state transitions are valid.
func (ac *addrConn) resetTransport(drain bool) error {
	ac.mu.Lock()
	if ac.state == connectivity.Shutdown {
		ac.mu.Unlock()
		return errConnClosing
	}
	oldState := ac.state
	ac.state = connectivity.Connecting
	ac.csEvltr.recordTransition(oldState, ac.state)
	if ac.cc.balancer != nil {
		ac.cc.balancer.HandleSubConnStateChange(ac.acbw, ac.state)
	}
	// TODO(bar) don't call balancer functions to handle subconn state change if ac.acbw is nil.
	if ac.ready != nil {
		close(ac.ready)
		ac.ready = nil
	}
	t := ac.transport
	ac.transport = nil
	ac.mu.Unlock()
	if t != nil && !drain {
		t.Close()
	}
	ac.cc.mu.RLock()
	ac.dopts.copts.KeepaliveParams = ac.cc.mkp
	ac.cc.mu.RUnlock()
	for retries := 0; ; retries++ {
		sleepTime := ac.dopts.bs.backoff(retries)
		timeout := minConnectTimeout
		ac.mu.Lock()
		if timeout < time.Duration(int(sleepTime)/len(ac.addrs)) {
			timeout = time.Duration(int(sleepTime) / len(ac.addrs))
		}
		connectTime := time.Now()
		if ac.state == connectivity.Shutdown {
			ac.mu.Unlock()
			return errConnClosing
		}
		ac.printf("connecting")
		oldState := ac.state
		ac.state = connectivity.Connecting
		ac.csEvltr.recordTransition(oldState, ac.state)
		// TODO(bar) remove condition once we always have a balancer.
		if ac.cc.balancer != nil {
			ac.cc.balancer.HandleSubConnStateChange(ac.acbw, ac.state)
		}
		// copy ac.addrs in case of race
		addrsIter := make([]resolver.Address, len(ac.addrs))
		copy(addrsIter, ac.addrs)
		ac.mu.Unlock()
		for _, addr := range addrsIter {
			ac.mu.Lock()
			if ac.state == connectivity.Shutdown {
				// ac.tearDown(...) has been invoked.
				ac.mu.Unlock()
				return errConnClosing
			}
			ac.mu.Unlock()
			ctx, cancel := context.WithTimeout(ac.ctx, timeout)
			sinfo := transport.TargetInfo{
				Addr:     addr.Addr,
				Metadata: addr.Metadata,
			}
			newTransport, err := transport.NewClientTransport(ctx, sinfo, ac.dopts.copts)
			// Don't call cancel in success path due to a race in Go 1.6:
			// https://github.com/golang/go/issues/15078.
			if err != nil {
				cancel()

				if e, ok := err.(transport.ConnectionError); ok && !e.Temporary() {
					return err
				}
				grpclog.Warningf("grpc: addrConn.resetTransport failed to create client transport: %v; Reconnecting to %v", err, addr)
				ac.mu.Lock()
				if ac.state == connectivity.Shutdown {
					// ac.tearDown(...) has been invoked.
					ac.mu.Unlock()
					return errConnClosing
				}
				ac.mu.Unlock()
				continue
			}
			ac.mu.Lock()
			ac.printf("ready")
			if ac.state == connectivity.Shutdown {
				// ac.tearDown(...) has been invoked.
				ac.mu.Unlock()
				newTransport.Close()
				return errConnClosing
			}
			oldState = ac.state
			ac.state = connectivity.Ready
			ac.csEvltr.recordTransition(oldState, ac.state)
			if ac.cc.balancer != nil {
				ac.cc.balancer.HandleSubConnStateChange(ac.acbw, ac.state)
			}
			ac.transport = newTransport
			ac.curAddr = addr
			if ac.ready != nil {
				close(ac.ready)
				ac.ready = nil
			}
			ac.mu.Unlock()
			return nil
		}
		ac.mu.Lock()
		oldState = ac.state
		ac.state = connectivity.TransientFailure
		ac.csEvltr.recordTransition(oldState, ac.state)
		if ac.cc.balancer != nil {
			ac.cc.balancer.HandleSubConnStateChange(ac.acbw, ac.state)
		}
		if ac.ready != nil {
			close(ac.ready)
			ac.ready = nil
		}
		ac.mu.Unlock()
		timer := time.NewTimer(sleepTime - time.Since(connectTime))
		select {
		case <-timer.C:
		case <-ac.ctx.Done():
			timer.Stop()
			return ac.ctx.Err()
		}
		timer.Stop()
	}
}

// Run in a goroutine to track the error in transport and create the
// new transport if an error happens. It returns when the channel is closing.
func (ac *addrConn) transportMonitor() {
	for {
		ac.mu.Lock()
		t := ac.transport
		ac.mu.Unlock()
		select {
		// This is needed to detect the teardown when
		// the addrConn is idle (i.e., no RPC in flight).
		case <-ac.ctx.Done():
			select {
			case <-t.Error():
				t.Close()
			default:
			}
			return
		case <-t.GoAway():
			ac.adjustParams(t.GetGoAwayReason())
			// If GoAway happens without any network I/O error, the underlying transport
			// will be gracefully closed, and a new transport will be created.
			// (The transport will be closed when all the pending RPCs finished or failed.)
			// If GoAway and some network I/O error happen concurrently, the underlying transport
			// will be closed, and a new transport will be created.
			var drain bool
			select {
			case <-t.Error():
			default:
				drain = true
			}
			if err := ac.resetTransport(drain); err != nil {
				grpclog.Infof("get error from resetTransport %v, transportMonitor returning", err)
				if err != errConnClosing {
					// Keep this ac in cc.conns, to get the reason it's torn down.
					ac.tearDown(err)
				}
				return
			}
		case <-t.Error():
			select {
			case <-ac.ctx.Done():
				t.Close()
				return
			case <-t.GoAway():
				ac.adjustParams(t.GetGoAwayReason())
			default:
			}
			if err := ac.resetTransport(false); err != nil {
				grpclog.Infof("get error from resetTransport %v, transportMonitor returning", err)
				ac.mu.Lock()
				ac.printf("transport exiting: %v", err)
				ac.mu.Unlock()
				grpclog.Warningf("grpc: addrConn.transportMonitor exits due to: %v", err)
				if err != errConnClosing {
					// Keep this ac in cc.conns, to get the reason it's torn down.
					ac.tearDown(err)
				}
				return
			}
		}
	}
}

// wait blocks until i) the new transport is up or ii) ctx is done or iii) ac is closed or
// iv) transport is in connectivity.TransientFailure and there is a balancer/failfast is true.
func (ac *addrConn) wait(ctx context.Context, hasBalancer, failfast bool) (transport.ClientTransport, error) {
	for {
		ac.mu.Lock()
		switch {
		case ac.state == connectivity.Shutdown:
			if failfast || !hasBalancer {
				// RPC is failfast or balancer is nil. This RPC should fail with ac.tearDownErr.
				err := ac.tearDownErr
				ac.mu.Unlock()
				return nil, err
			}
			ac.mu.Unlock()
			return nil, errConnClosing
		case ac.state == connectivity.Ready:
			ct := ac.transport
			ac.mu.Unlock()
			return ct, nil
		case ac.state == connectivity.TransientFailure:
			if failfast || hasBalancer {
				ac.mu.Unlock()
				return nil, errConnUnavailable
			}
		}
		ready := ac.ready
		if ready == nil {
			ready = make(chan struct{})
			ac.ready = ready
		}
		ac.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, toRPCErr(ctx.Err())
		// Wait until the new transport is ready or failed.
		case <-ready:
		}
	}
}

// tearDown starts to tear down the addrConn.
// TODO(zhaoq): Make this synchronous to avoid unbounded memory consumption in
// some edge cases (e.g., the caller opens and closes many addrConn's in a
// tight loop.
// tearDown doesn't remove ac from ac.cc.conns.
func (ac *addrConn) tearDown(err error) {
	ac.cancel()

	ac.mu.Lock()
	ac.curAddr = resolver.Address{}
	defer ac.mu.Unlock()
	if err == errConnDrain && ac.transport != nil {
		// GracefulClose(...) may be executed multiple times when
		// i) receiving multiple GoAway frames from the server; or
		// ii) there are concurrent name resolver/Balancer triggered
		// address removal and GoAway.
		ac.transport.GracefulClose()
	}
	if ac.state == connectivity.Shutdown {
		return
	}
	oldState := ac.state
	ac.state = connectivity.Shutdown
	ac.tearDownErr = err
	ac.csEvltr.recordTransition(oldState, ac.state)
	if ac.cc.balancer != nil {
		ac.cc.balancer.HandleSubConnStateChange(ac.acbw, ac.state)
	}
	if ac.events != nil {
		ac.events.Finish()
		ac.events = nil
	}
	if ac.ready != nil {
		close(ac.ready)
		ac.ready = nil
	}
	if ac.transport != nil && err != errConnDrain {
		ac.transport.Close()
	}
	return
}

func (ac *addrConn) getState() connectivity.State {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return ac.state
}
