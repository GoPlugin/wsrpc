package uni_test

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/goplugin/wsrpc"
	pb "github.com/goplugin/wsrpc/intgtest/internal/rpcs"
	"github.com/goplugin/wsrpc/intgtest/utils"
	"github.com/goplugin/wsrpc/peer"
)

func Test_ServerClient_SimpleCall(t *testing.T) {
	keypairs := utils.GenerateKeys(t)
	pubKeys := []ed25519.PublicKey{keypairs.Client1.PubKey}

	// Start the server
	lis, s := utils.SetupServer(t,
		wsrpc.WithCreds(keypairs.Server.PrivKey, pubKeys),
	)

	// Start serving
	go s.Serve(lis)
	t.Cleanup(s.Stop)
	// Create an RPC client for the server
	c := pb.NewEchoClient(s)

	// Start client
	conn, err := utils.SetupClientConnWithOptsAndTimeout(t, 5*time.Second,
		wsrpc.WithTransportCreds(keypairs.Client1.PrivKey, keypairs.Server.PubKey),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	// Register the handlers on the wsrpc client
	pb.RegisterEchoServer(conn, &utils.EchoServer{})

	// Wait for the connection to be established
	utils.WaitForReadyConnection(t, conn)

	ctx := peer.NewCallContext(context.Background(), keypairs.Client1.StaticallySizedPublicKey(t))
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := c.Echo(ctx, &pb.EchoRequest{Body: "bodyarg"})
	require.NoError(t, err)

	assert.Equal(t, "bodyarg", resp.Body)
}

func Test_ServerClient_ConcurrentCalls(t *testing.T) {
	keypairs := utils.GenerateKeys(t)
	pubKeys := []ed25519.PublicKey{keypairs.Client1.PubKey}

	// Start the serverTest_ServerClient_ConcurrentCalls
	lis, s := utils.SetupServer(t,
		wsrpc.WithCreds(keypairs.Server.PrivKey, pubKeys),
	)

	// Register the ping server implementation with the wsrpc server
	pb.RegisterEchoServer(s, &utils.EchoServer{})
	// Create an RPC client for the server
	c := pb.NewEchoClient(s)

	// Start serving
	go s.Serve(lis)
	t.Cleanup(s.Stop)

	// Start client
	conn, err := utils.SetupClientConnWithOptsAndTimeout(t, 500*time.Second,
		wsrpc.WithTransportCreds(keypairs.Client1.PrivKey, keypairs.Server.PubKey),
		wsrpc.WithBlock(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	// Register the handlers on the wsrpc client
	pb.RegisterEchoServer(conn, &utils.EchoServer{})

	respCh := make(chan *pb.EchoResponse)
	doneCh := make(chan []*pb.EchoResponse)

	pk := keypairs.Client1.StaticallySizedPublicKey(t)
	reqs := []utils.EchoReq{
		{Message: &pb.EchoRequest{Body: "call1", DelayMs: 500}, PubKey: &pk},
		{Message: &pb.EchoRequest{Body: "call2"}, Timeout: 2000 * time.Millisecond, PubKey: &pk},
	}

	go func() {
		doneCh <- utils.WaitForResponses(t, respCh, len(reqs))
	}()

	utils.ProcessEchos(t, c, reqs, respCh)

	actual := <-doneCh

	assert.Equal(t, len(reqs), len(actual))
	assert.Equal(t, "call2", actual[0].Body)
	assert.Equal(t, "call1", actual[1].Body)
}
