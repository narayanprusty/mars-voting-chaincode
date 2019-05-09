package main

import (
	"fmt"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/util"
	"crypto"
  "crypto/rsa"
  "crypto/x509"
  "encoding/base64"
	"encoding/pem"
	"strconv"
)

type VotingChaincode struct {
}

type User struct {
	PublicKey	string `json:"publicKey"`
	MetadataHash string `json:"metadataHash"`
	Permissions []string `json:"permissions"`
}

func (t *VotingChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	var err error
	var identity []byte

	identity, err = stub.GetCreator()
	
	if err != nil {
		return shim.Error("An error occured")
	}

	sId := &msp.SerializedIdentity{}
	err = proto.Unmarshal(identity, sId)
	
	if err != nil {
			return shim.Error("An error occured")
	}

	nodeId := sId.Mspid
	err = stub.PutState("votingAuthority", []byte(nodeId))

	if err != nil {
		return shim.Error("An error occured")
	}

	return shim.Success(nil)
}

func (t *VotingChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "getCreatorIdentity" {
		return t.getCreatorIdentity(stub, args)
	} else if function == "vote" {
		return t.vote(stub, args)
	} else if function == "getVotes" {
		return t.getVotes(stub, args)
	}

	return shim.Error("Invalid function name: " + function)
}

func (t *VotingChaincode) getCreatorIdentity(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	identity, err := stub.GetState("votingAuthority")

	if err != nil {
		return shim.Error("An error occured")
	}

	if identity == nil {
		return shim.Error("Identity not yet stored")
	}

	return shim.Success(identity)
}

func (t *VotingChaincode) getVotes(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments.")
	}

	var err error

	to := args[0]

	votes, err := stub.GetState(to)

	if err != nil {
		return shim.Error("An error occured while reading votes")
	}

	return shim.Success(votes)
}

func (t *VotingChaincode) vote(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments.")
	}

	var err error

	user := args[0]

	tMap, err := stub.GetTransient()

	to := string(tMap["to"])
	signature := tMap["signature"]
	signature, err = base64.StdEncoding.DecodeString(string(signature))

	if err != nil {
		return shim.Error("An error occured")
	}

	identityChannelName := args[1]

	identity, err := stub.GetCreator()

	if err != nil {
		return shim.Error("An error occured")
	}

	sId := &msp.SerializedIdentity{}
	err = proto.Unmarshal(identity, sId)
	
	if err != nil {
		return shim.Error("An error occured")
	}

	votingAuthority, err := stub.GetState("votingAuthority")

	nodeId := sId.Mspid

	if string(votingAuthority) != nodeId {
		return shim.Error("You are not authorized")
	}

	userVoted, err := stub.GetState("voted_" + user)

	if(userVoted != nil) {
		return shim.Error("User has already voted")
	}

	votes, err := stub.GetState(to)

	if err != nil {
		return shim.Error("An error occured while fetching votes")
	}

	if votes == nil {
		votes = []byte("0")
	}

	count, err := strconv.Atoi(string(votes))

	if err != nil {
		return shim.Error("An error occured ")
	}

	chainCodeArgs := util.ToChaincodeArgs("getIdentity", user)
	response := stub.InvokeChaincode("identity", chainCodeArgs, identityChannelName)

	if response.Status != shim.OK {
		return shim.Error(response.Message)
 	}

	var userStruct User
	err = json.Unmarshal(response.Payload, &userStruct)

	if err != nil {
		return shim.Error("User struct creation failed")
	}

	userPublicKey, err := base64.StdEncoding.DecodeString(userStruct.PublicKey)

	fmt.Printf("Message: %s", string(userStruct.PublicKey))

	block, _ := pem.Decode(userPublicKey)

	if block == nil {
    return shim.Error("Pem decoded")
	}
	
	userPublicKeyObj, err := x509.ParsePKIXPublicKey(block.Bytes)

	if err != nil {
		return shim.Error("Public key invalid")
	}

	message := []byte("{\"action\":\"vote\",\"to\":\"" + to + "\"}")

	fmt.Printf("Message: %s", string(message))

	newhash := crypto.SHA256
  pssh := newhash.New()
  pssh.Write(message)
	hashed := pssh.Sum(nil)
	
	rsaPublickey, _ := userPublicKeyObj.(*rsa.PublicKey)
	
	err = rsa.VerifyPKCS1v15(rsaPublickey, crypto.SHA256, hashed, signature)

	if err != nil {
		return shim.Error("Signature invalid")
	}

	count++

	fmt.Printf("New votes: %s", count)

	err = stub.PutState(to, []byte(strconv.Itoa(count)))

	if err != nil {
		return shim.Error("An error occured while writing votes")
	}

	err = stub.PutState("voted_" + user, []byte("true"))

	if err != nil {
		return shim.Error("An error occured while writing state")
	}
	
	return shim.Success(nil)
}


func main() {
	err := shim.Start(new(VotingChaincode))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
