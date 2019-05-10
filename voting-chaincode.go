package main

import (
	"fmt"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/util"
	"github.com/decred/dcrd/dcrec/secp256k1"
	"encoding/hex"
	"crypto/sha256"
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
		return shim.Error(err.Error())
	}

	sId := &msp.SerializedIdentity{}
	err = proto.Unmarshal(identity, sId)
	
	if err != nil {
			return shim.Error(err.Error())
	}

	nodeId := sId.Mspid
	err = stub.PutState("votingAuthority", []byte(nodeId))

	if err != nil {
		return shim.Error(err.Error())
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
		return shim.Error(err.Error())
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
		return shim.Error(err.Error())
	}

	return shim.Success(votes)
}

func (t *VotingChaincode) vote(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments.")
	}

	var err error

	user := args[0]

	tMap, err := stub.GetTransient()

	to := string(tMap["to"])

	if err != nil {
		return shim.Error(err.Error())
	}

	identityChannelName := args[1]

	identity, err := stub.GetCreator()

	if err != nil {
		return shim.Error(err.Error())
	}

	sId := &msp.SerializedIdentity{}
	err = proto.Unmarshal(identity, sId)
	
	if err != nil {
		return shim.Error(err.Error())
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
		return shim.Error(err.Error())
	}

	if votes == nil {
		votes = []byte("0")
	}

	count, err := strconv.Atoi(string(votes))

	if err != nil {
		return shim.Error(err.Error())
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

	pubKeyBytes, err := hex.DecodeString(userStruct.PublicKey)
	if err != nil {
		return shim.Error(err.Error())
	}

	pubKey, err := secp256k1.ParsePubKey(pubKeyBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	sigBytes, err := hex.DecodeString(args[2])

	if err != nil {
		return shim.Error(err.Error())
	}

	signature, err := secp256k1.ParseDERSignature(sigBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	message := []byte("{\"action\":\"vote\",\"to\":\"" + to + "\"}")

	messageHash := sha256.Sum256([]byte(message))
	verified := signature.Verify(messageHash[:], pubKey)
	
	if (verified) {
		count++

		fmt.Printf("New votes: %s", count)
	
		err = stub.PutState(to, []byte(strconv.Itoa(count)))
	
		if err != nil {
			return shim.Error(err.Error())
		}
	
		err = stub.PutState("voted_" + user, []byte("true"))
	
		if err != nil {
			return shim.Error(err.Error())
		}		
	} else {
		return shim.Error("Invalid signature")
	}

	return shim.Success(nil)
}


func main() {
	err := shim.Start(new(VotingChaincode))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
