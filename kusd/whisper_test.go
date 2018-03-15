package kusd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kowala-tech/kUSD/common/hexutil"
	"github.com/kowala-tech/kUSD/core"
	"github.com/kowala-tech/kUSD/node"
	"github.com/kowala-tech/kUSD/p2p"
	"github.com/kowala-tech/kUSD/p2p/discover"
	"github.com/kowala-tech/kUSD/rpc"
	"github.com/kowala-tech/kUSD/whisper"
)

func TestSendMessage_Success(t *testing.T) {
	//Start nodes
	fullnode, stop := startNode("fullNode", false, t)
	defer stop()

	sender, stop := startNode("sender", false, t)
	defer stop()

	receiver, stop := startNode("receiver", false, t)
	defer stop()

	// add full node to all nodes
	err := addPeer(fullnode, sender)
	if err != nil {
		t.Fatalf("can't add full node to sender's peers: %v", err)
	}

	err = addPeer(fullnode, receiver)
	if err != nil {
		t.Fatalf("can't add full node to receiver's peers: %v", err)
	}

	//some time to finish adding peers
	time.Sleep(1 * time.Second)

	//create a topic
	topic := whisper.BytesToTopic([]byte("topic name"))
	const password = "some password"

	//make a key
	senderSymKeyID, err := sender.Shh().AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create sender's key: %v", err)
	}

	receiverSymKeyID, err := receiver.Shh().AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create receiver's key: %v", err)
	}

	//Create message a filter
	messageFilterID, err := createChatMessageFilter(sender.Client, senderSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//There are no messages at the filter
	messages, err := getMessagesByMessageFilterID(sender.Client, messageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatal("filter should be empty on the start")
	}

	receiverMessageFilterID, err := createChatMessageFilter(receiver.Client, receiverSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//Post message matching with filter (key and token)
	messageText := hexutil.Encode([]byte("Hello world!"))
	err = postMessage(sender.Client, senderSymKeyID, topic.String(), messageText)
	if err != nil {
		t.Fatal(err)
	}

	//act

	time.Sleep(1 * time.Second)

	messages, err = getMessagesByMessageFilterID(receiver.Client, receiverMessageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatal("filter should received 1 message")
	}

	if messages[0]["payload"].(string) != messageText {
		t.Fatalf("got wrong message: %s", messages[0]["payload"].(string))
	}
}

func TestSendMessageWithLightNodes_Success(t *testing.T) {
	//Start nodes
	fullnode, stop := startNode("fullNode", false, t)
	defer stop()

	sender, stop := startNode("sender", true, t)
	defer stop()

	receiver, stop := startNode("receiver", true, t)
	defer stop()

	// add full node to all nodes
	err := addPeer(fullnode, sender)
	if err != nil {
		t.Fatalf("can't add full node to sender's peers: %v", err)
	}

	err = addPeer(fullnode, receiver)
	if err != nil {
		t.Fatalf("can't add full node to receiver's peers: %v", err)
	}

	//some time to finish adding peers
	time.Sleep(1 * time.Second)

	//create a topic
	topic := whisper.BytesToTopic([]byte("topic name"))
	const password = "some password"

	//make a key
	senderSymKeyID, err := sender.Shh().AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create sender's key: %v", err)
	}

	receiverSymKeyID, err := receiver.Shh().AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create receiver's key: %v", err)
	}

	//Create message a filter
	messageFilterID, err := createChatMessageFilter(sender.Client, senderSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//There are no messages at the filter
	messages, err := getMessagesByMessageFilterID(sender.Client, messageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatal("filter should be empty on the start")
	}

	receiverMessageFilterID, err := createChatMessageFilter(receiver.Client, receiverSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//Post message matching with filter (key and token)
	messageText := hexutil.Encode([]byte("Hello world!"))
	err = postMessage(sender.Client, senderSymKeyID, topic.String(), messageText)
	if err != nil {
		t.Fatal(err)
	}

	//act

	time.Sleep(1 * time.Second)

	messages, err = getMessagesByMessageFilterID(receiver.Client, receiverMessageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatalf("filter should received 1 message. got %d", len(messages))
	}

	if messages[0]["payload"].(string) != messageText {
		t.Fatalf("got wrong message: %s", messages[0]["payload"].(string))
	}
}

func TestSendMessageWrongKey_Fail(t *testing.T) {
	//Start nodes
	fullnode, stop := startNode("fullNode", false, t)
	defer stop()

	sender, stop := startNode("sender", false, t)
	defer stop()

	receiver, stop := startNode("receiver", false, t)
	defer stop()

	// add full node to all nodes
	err := addPeer(fullnode, sender)
	if err != nil {
		t.Fatalf("can't add full node to sender's peers: %v", err)
	}

	err = addPeer(fullnode, receiver)
	if err != nil {
		t.Fatalf("can't add full node to receiver's peers: %v", err)
	}

	//some time to finish adding peers
	time.Sleep(1 * time.Second)

	//create a topic
	topic := whisper.BytesToTopic([]byte("topic name"))
	const password = "some password"
	const incorrectPassword = ""

	//make a key
	senderSymKeyID, err := sender.Shh().AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create sender's key: %v", err)
	}

	receiverSymKeyID, err := receiver.Shh().AddSymKeyFromPassword(incorrectPassword)
	if err != nil {
		t.Fatalf("failed to create receiver's key: %v", err)
	}

	//Create message a filter
	messageFilterID, err := createChatMessageFilter(sender.Client, senderSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//There are no messages at the filter
	messages, err := getMessagesByMessageFilterID(sender.Client, messageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatal("filter should be empty on the start")
	}

	receiverMessageFilterID, err := createChatMessageFilter(receiver.Client, receiverSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//Post message matching with filter (key and token)
	messageText := hexutil.Encode([]byte("Hello world!"))
	err = postMessage(sender.Client, senderSymKeyID, topic.String(), messageText)
	if err != nil {
		t.Fatal(err)
	}

	//act

	time.Sleep(1 * time.Second)

	messages, err = getMessagesByMessageFilterID(receiver.Client, receiverMessageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatal("filter shouldn't receive any messages")
	}
}

func TestSendMessageOnlyLightPeers_Fail(t *testing.T) {
	//Start nodes
	fullnode, stop := startNode("fullNode", true, t)
	defer stop()

	sender, stop := startNode("sender", true, t)
	defer stop()

	receiver, stop := startNode("receiver", true, t)
	defer stop()

	// add full node to all nodes
	err := addPeer(fullnode, sender)
	if err != nil {
		t.Fatalf("can't add full node to sender's peers: %v", err)
	}

	err = addPeer(fullnode, receiver)
	if err != nil {
		t.Fatalf("can't add full node to receiver's peers: %v", err)
	}

	//some time to finish adding peers
	time.Sleep(1 * time.Second)

	//create a topic
	topic := whisper.BytesToTopic([]byte("topic name"))
	const password = "some password"

	//make a key
	senderSymKeyID, err := sender.Shh().AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create sender's key: %v", err)
	}

	receiverSymKeyID, err := receiver.Shh().AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create receiver's key: %v", err)
	}

	//Create message a filter
	messageFilterID, err := createChatMessageFilter(sender.Client, senderSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//There are no messages at the filter
	messages, err := getMessagesByMessageFilterID(sender.Client, messageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatal("filter should be empty on the start")
	}

	receiverMessageFilterID, err := createChatMessageFilter(receiver.Client, receiverSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//act

	//Post message matching with filter (key and token)
	messageText := hexutil.Encode([]byte("Hello world!"))
	err = postMessage(sender.Client, senderSymKeyID, topic.String(), messageText)
	if err.Error() != fmt.Sprintf("-32000 %s", whisper.ErrNoFullPeers.Error()) {
		t.Fatalf("got whong error %v", err)
	}

	time.Sleep(1 * time.Second)

	messages, err = getMessagesByMessageFilterID(receiver.Client, receiverMessageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatal("filter should't receive any messages")
	}
}

func TestSendMessageNoPeers_Fail(t *testing.T) {
	//Start nodes
	sender, stop := startNode("sender", false, t)
	defer stop()

	receiver, stop := startNode("receiver", false, t)
	defer stop()

	//some time to finish adding peers
	time.Sleep(1 * time.Second)

	//create a topic
	topic := whisper.BytesToTopic([]byte("topic name"))
	const password = "some password"

	//make a key
	senderSymKeyID, err := sender.Shh().AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create sender's key: %v", err)
	}

	receiverSymKeyID, err := receiver.Shh().AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed to create receiver's key: %v", err)
	}

	//Create message a filter
	messageFilterID, err := createChatMessageFilter(sender.Client, senderSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//There are no messages at the filter
	messages, err := getMessagesByMessageFilterID(sender.Client, messageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatal("filter should be empty on the start")
	}

	receiverMessageFilterID, err := createChatMessageFilter(receiver.Client, receiverSymKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	//act

	//Post message matching with filter (key and token)
	messageText := hexutil.Encode([]byte("Hello world!"))
	err = postMessage(sender.Client, senderSymKeyID, topic.String(), messageText)
	if err.Error() != fmt.Sprintf("-32000 %s", whisper.ErrNoPeers.Error()) {
		t.Fatalf("got wrong error %v", err)
	}

	time.Sleep(1 * time.Second)

	messages, err = getMessagesByMessageFilterID(receiver.Client, receiverMessageFilterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatal("filter should't receive any messages")
	}
}

var nodesCount uint32

func startNode(name string, isLight bool, t *testing.T) (*runningNode, func()) {
	dir, err := ioutil.TempDir("", name)
	if err != nil {
		t.Fatalf("failed to create manual data dir: %v", err)
	}

	nodeNumber := int(atomic.AddUint32(&nodesCount, 1))
	listenAddr := 22334 + nodeNumber

	config := &node.Config{
		SHH:	  true,
		SHHLight: isLight,
		DataDir:  dir,
		HTTPPort: 8645 + nodeNumber,
		WSPort:   8646 + nodeNumber,
		Name:     name,
		P2P: p2p.Config{
			MaxPeers:        25,
			MaxPendingPeers: 25,
			ListenAddr:      ":" + strconv.Itoa(listenAddr),
		},
	}

	stack, err := node.New(config)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	kusdConf := &Config{
		NetworkId: 3,
		Genesis:   core.DefaultTestnetGenesisBlock(),
	}

	if err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) { return New(ctx, kusdConf) }); err != nil {
		t.Fatalf("failed to register Kowala protocol: %v", err)
	}

	//fixme: we should create method like IsNodeRunning
	if stack.Server() != nil {
		t.Fatal("node is running before Start has been called")
	}

	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}

	if stack.Server() == nil {
		t.Fatal("node is not running after Start has been called")
	}

	rpcClient, err := newRPCClient(stack)
	if rpcClient == nil {
		t.Fatalf("node RPC client is not running after Start has been called: %v", err)
	}

	// Create the final tester and return
	var kowala *Kowala
	stack.Service(&kowala)

	return &runningNode{kowala, stack, rpcClient}, func() {
		if stack.Server() == nil {
			t.Fatal("node is not running after Start has been called")
		}

		if err := stack.Stop(); err != nil {
			t.Fatalf("failed to stop node: %v", err)
		}

		if stack.Server() != nil {
			t.Fatal("node is running after Stop has been called")
		}

		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("failed to clean node directory: %v", err)
		}
	}
}

//createChatMessageFilter create message filter with symmetric encryption
func createChatMessageFilter(rpcCli *rpc.Client, symKeyID string, topic string) (string, error) {
	resp := rpcCli.CallRaw(`{
			"jsonrpc": "2.0",
			"method": "shh_newMessageFilter", "params": [
				{"symKeyID": "` + symKeyID + `", "topics": [ "` + topic + `"], "allowP2P":true}
			],
			"id": 1
		}`)

	msgFilterResp := returnedIDResponse{}
	if err := json.Unmarshal([]byte(resp), &msgFilterResp); err != nil {
		return "", fmt.Errorf("can't unmarshall creating message filter responce: %v", err.Error())
	}

	if msgFilterResp.Error != nil {
		return "", fmt.Errorf("can't create message filter responce: %v", msgFilterResp.Error)
	}

	messageFilterID := msgFilterResp.Result

	if messageFilterID == "" {
		return "", fmt.Errorf("create message filter returns empty responce: %v", resp)
	}

	return messageFilterID, nil
}

func postMessage(rpcCli *rpc.Client, symKeyID string, topic string, payload string) error {
	resp := rpcCli.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"symKeyID": "` + symKeyID + `",
			"topic": "` + topic + `",
			"payload": "` + payload + `",
			"powTarget": 0.2,
			"powTime": 1
			}
		],
		"id": 1}`)

	postResp := baseRPCResponse{}

	if err := json.Unmarshal([]byte(resp), &postResp); err != nil {
		return fmt.Errorf("can't unmarshall post message responce: %v", err.Error())
	}

	if postResp.Error != nil {
		return *postResp.Error
	}

	return nil
}

//getMessagesByMessageFilterID get received messages by messageFilterID
func getMessagesByMessageFilterID(rpcCli *rpc.Client, messageFilterID string) ([]map[string]interface{}, error) {
	resp := rpcCli.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	messages := getFilterMessagesResponse{}

	if err := json.Unmarshal([]byte(resp), &messages); err != nil {
		return nil, fmt.Errorf("can't get messages by the filter: %v", err.Error())
	}

	if messages.Error != nil {
		return nil, fmt.Errorf("can't get messages by the filter: %v", messages.Error)
	}

	return messages.Result, nil
}

// init RPC client for the node
func newRPCClient(n *node.Node) (*rpc.Client, error) {
	localRPCClient, err := n.Attach()
	if err != nil {
		return nil, err
	}

	return localRPCClient, nil
}

// addPeer adds new static peer node
func addPeer(src, dst *runningNode) error {
	enodeToAdd := src.Server().NodeInfo().Enode
	parsedNode, err := discover.ParseNode(enodeToAdd)
	if err != nil {
		return err
	}

	dst.Server().AddPeer(parsedNode)
	return nil
}

type getFilterMessagesResponse struct {
	Result []map[string]interface{}
	Error  *Error
}

type returnedIDResponse struct {
	Result string
	Error  *Error
}
type baseRPCResponse struct {
	Result interface{}
	Error  *Error
}

type Error struct {
	Code int
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Message)
}

type runningNode struct {
	*Kowala
	*node.Node
	*rpc.Client
}
