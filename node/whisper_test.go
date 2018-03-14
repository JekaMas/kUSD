package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/kowala-tech/kUSD/common/hexutil"
	"github.com/kowala-tech/kUSD/crypto"
	"github.com/kowala-tech/kUSD/rpc"
	"github.com/kowala-tech/kUSD/whisper"
)

func TestSendMessage(t *testing.T) {
	//Start nodes
	sender, stop := startNode("sender", t)
	defer stop()

	receiver, stop := startNode("receiver", t)
	defer stop()
	_ = receiver

	//create a topic
	topic := whisper.BytesToTopic([]byte("topic name"))

	//Add a key pair to whisper
	keyID, err := sender.Shh().NewKeyPair()
	if err != nil {
		t.Fatalf("failed to create key pair: %v", err)
	}

	key, err := sender.Shh().GetPrivateKey(keyID)
	if err != nil {
		t.Fatalf("failed to get private key: %v", err)
	}

	pubkey := hexutil.Bytes(crypto.FromECDSAPub(&key.PublicKey))

	//Create message a filter
	messageFilterID, err := createPrivateChatMessageFilter(sender.Client, keyID, topic.String())
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

	//Post message matching with filter (key and token)
	messageText := hexutil.Encode([]byte("Hello world!"))
	err = postMessageToPrivate(sender.Client, pubkey.String(), topic.String(), messageText)
	if err != nil {
		t.Fatal(err)
	}

	//act

	//Get message to make sure that it will come from the mailbox later
	time.Sleep(1 * time.Second)

	messages, err = getMessagesByMessageFilterID(sender.Client, messageFilterID)
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

//Start node
func startNode(name string, t *testing.T) (*runningNode, func()) {
	dir, err := ioutil.TempDir("", name)
	if err != nil {
		t.Fatalf("failed to create manual data dir: %v", err)
	}

	if _, err := New(&Config{DataDir: dir}); err != nil {
		t.Fatalf("failed to create stack with existing datadir: %v", err)
	}

	config := testNodeConfig()
	config.DataDir = dir

	stack, err := New(config)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}

	//fixme: we should create method like IsNodeRunning
	if stack.server != nil {
		t.Fatal("node is running before Start has been called")
	}

	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}

	if stack.server == nil {
		t.Fatal("node is not running after Start has been called")
	}

	rpcClient, err := newRPCClient(stack)
	if rpcClient == nil {
		t.Fatalf("node RPC client is not running after Start has been called: %v", err)
	}

	return &runningNode{stack, rpcClient}, func() {
		if stack.server == nil {
			t.Fatalf("node is not running after Start has been called")
		}

		if err := stack.Stop(); err != nil {
			t.Fatalf("failed to stop node: %v", err)
		}

		if stack.server != nil {
			t.Fatalf("node is running after Stop has been called")
		}

		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("failed to clean node directory: %v", err)
		}
	}

}

//createPrivateChatMessageFilter create message filter with asymmetric encryption
func createPrivateChatMessageFilter(rpcCli *rpc.Client, privateKeyID string, topic string) (string, error) {
	resp := rpcCli.CallRaw(`{
			"jsonrpc": "2.0",
			"method": "shh_newMessageFilter", "params": [
				{"privateKeyID": "` + privateKeyID + `", "topics": [ "` + topic + `"], "allowP2P":true}
			],
			"id": 1
		}`)

	msgFilterResp := returnedIDResponse{}
	if err := json.Unmarshal([]byte(resp), &msgFilterResp); err != nil {
		return "", fmt.Errorf("can't unmarshall creating message filter responce: %v", err.Error())
	}

	if msgFilterResp.Err != nil {
		return "", fmt.Errorf("can't create message filter responce: %v", msgFilterResp.Err)
	}

	messageFilterID := msgFilterResp.Result

	if messageFilterID == "" {
		return "", fmt.Errorf("create message filter returns empty responce: %v", resp)
	}

	return messageFilterID, nil
}

func postMessageToPrivate(rpcCli *rpc.Client, bobPubkey string, topic string, payload string) error {
	resp := rpcCli.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"pubKey": "` + bobPubkey + `",
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

	if postResp.Err != nil {
		return fmt.Errorf("can't post the message responce: %v", postResp.Err)
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

	if messages.Err != nil {
		return nil, fmt.Errorf("can't get messages by the filter: %v", messages.Err)
	}

	return messages.Result, nil
}

// init RPC client for the node
func newRPCClient(n *Node) (*rpc.Client, error) {
	localRPCClient, err := n.Attach()
	if err != nil {
		return nil, err
	}

	return localRPCClient, nil
}

type getFilterMessagesResponse struct {
	Result []map[string]interface{}
	Err    interface{}
}

type returnedIDResponse struct {
	Result string
	Err    interface{}
}
type baseRPCResponse struct {
	Result interface{}
	Err    interface{}
}

type runningNode struct {
	*Node
	*rpc.Client
}
