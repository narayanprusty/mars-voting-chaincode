const shim = require('fabric-shim');
const ethGSV = require('ethereum-gen-sign-verify');
var ab2str = require('arraybuffer-to-string')

var Chaincode = class {

  // Initialize the chaincode
  async Init(stub) {
    try {
      await stub.putState("votingAuthority", Buffer.from(stub.getCreator().mspid));
      return shim.success();
    } catch (err) {
      return shim.error(err);
    }
  }

  async Invoke(stub) {
    let ret = stub.getFunctionAndParameters();
    let method = this[ret.fcn];
    
    if (!method) {
      console.log('no method of name:' + ret.fcn + ' found');
      return shim.error();
    }

    try {
      let payload = await method(stub, ret.params);
      return shim.success(payload);
    } catch (err) {
      console.log(err);
      return shim.error(err);
    }
  }

  async getCreatorIdentity(stub) {
    let creatorIdentity = await stub.getState('votingAuthority');

    if (!creatorIdentity) {
      throw new Error("Creator identity not found");
    }

    return creatorIdentity;
  }

  async createVotingPhase(stub, args) {
    if (args.length < 1) {
      throw new Error('Incorrect number of arguments.');
    }

    let id = args[0]
    let votingPhase = {status: 'open', parties: {}, votes: {}}

    if(!(await stub.getState(`vp_${id}`)).toString()) {
      await stub.putState(`vp_${id}`, Buffer.from(JSON.stringify(votingPhase)))
    } else {
      throw new Error('Voting phase already exists');
    }
  }

  async getVotingPhase(stub, args) {
    if (args.length < 1) {
      throw new Error('Incorrect number of arguments.');
    }

    let votingPhase = await stub.getState(`vp_${args[0]}`);

    if(!votingPhase) {
      throw new Error('Voting Phase not found');
    }

    return votingPhase;
  }

  async vote(stub, args) {
    if (args.length < 2) {
      throw new Error('Incorrect number of arguments. ');
    }

    let secretData = stub.getTransient(); 
   
    let signature = {
      r: new Buffer(secretData.get('signature_r').toBuffer().toString('base64'), 'base64').toString('utf8'), 
      s: new Buffer(secretData.get('signature_s').toBuffer().toString('base64'), 'base64').toString('utf8'), 
      v: new Buffer(secretData.get('signature_v').toBuffer().toString('base64'), 'base64').toString('utf8') 
    };


    let message = {
      action: 'vote', 
      for: new Buffer(secretData.get('to').toBuffer().toString('base64'), 'base64').toString('utf8') 
    };

    if(!signature || !message) {
      throw new Error('Transient data missing');
    }

    let userId = args[0];
    let votingPhaseId = args[1];

    let result = await stub.invokeChaincode('identity', [
      Buffer.from('getIdentity'),
      Buffer.from(userId)
    ])

    if(result.status !== 200) {
      throw new Error('Internal transaction failed');
    }

    let publicKey = JSON.parse(result.payload.toString('utf8')).publicKey;

    if(message.action !== 'vote') {
      throw new Error('Permission invalid');
    }

    let isValid = ethGSV.verify(JSON.stringify(message), signature, publicKey);

    if(!isValid) {
      throw new Error('Signature invalid');
    }

    let votingPhase = (await stub.getState(`vp_${votingPhaseId}`)).toString();
    
    if(votingPhase) {
      votingPhase = JSON.parse(votingPhase)
      if(votingPhase.status === 'open') {
        if((await stub.getState('votingAuthority')).toString() === stub.getCreator().mspid) {
          if(!votingPhase.votes[userId]) {
            if(votingPhase.parties[message.for] === undefined) {
              votingPhase.parties[message.for] = 0;
            }
  
            votingPhase.parties[message.for] += 1
            votingPhase.votes[userId] = true

            await stub.putState(`vp_${votingPhaseId}`, Buffer.from(JSON.stringify(votingPhase)))
          } else {
            throw new Error('User has already voted');
          }
        } else {
          throw new Error('You don\'t have permission to vote');
        }
      } else {
        throw new Error('Voting phase is closed');
      }
    } else {
      throw new Error('Voting phase doesn\'t exist');
    }
  }

  async closePhase() {
    if (args.length < 1) {
      throw new Error('Incorrect number of arguments.');
    }

    let votingPhase = args[0]

    let votingPhase = (await stub.getState(`vp_${votingPhase}`)).toString();

    if(votingPhase) {
      votingPhase = JSON.parse(votingPhase)
      votingPhase.status = 'close'

      await stub.putState(`vp_${votingPhase}`, Buffer.from(JSON.stringify(votingPhase)))
    } else {
      throw new Error('Voting phase doesn\'t exist');
    }
  }
};

shim.start(new Chaincode());
