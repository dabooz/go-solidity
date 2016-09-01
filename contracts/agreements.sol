contract agreements {

    // The owner of this ethereum contract instance
    address owner;

    // This represents the state of an agreement
    struct finalAgreement {
        bytes32 contractHash;
        bytes32 agreementID;
        bytes   producerSig;
    }

    // This map ties the parties to a specific smarter contract
    // key is the agreement ID
    struct agreementInstance {
        mapping (bytes32 => finalAgreement) theAgreement;
    }

    // This map ties the counter party to the consumer
    // key is the counter party
    struct counter_party {
        mapping (address => agreementInstance) agreementInstances;
    }

    // This map holds all info for any consumer
    // key is the consumer's account address
    mapping (address => counter_party) allAgreements;

    // Events are emitted from this smart contract when important things happen. The event code
    // indicates which event occurred and the data associated with the event provides the context.
    enum event_codes {
        created,                           // An agreement was created
        created_detail,                    // The details of the created agreement
        create_fraud_alert,                // The caller tried to create a fraudulent agreement
        consumer_terminated,               // The consumer terminated the agreement
        producer_terminated,               // The producer terminated the agreement
        terminate_fraud_alert,             // The caller tried to terminate an agreement incorrectly
        admin_deleted                      // The ethereum smart contract owner deleted the agreement
    }

    event CreatedAgreement(uint    indexed _event_code,
                           address indexed _consumer,
                           address indexed _producer,
                           bytes32 indexed _agreementID) anonymous;

    event CreatedDetail(uint    indexed _event_code,
                        address indexed _consumer,
                        address indexed _producer,
                        bytes32 indexed _agreementID,
                        bytes32 _contractHash,
                        bytes  _producerSig) anonymous;

    event CreateFraudAlert(uint    indexed _event_code,
                           address indexed _consumer,
                           address indexed _producer,
                           bytes32 indexed _agreementID,
                           bytes32 _contractHash,
                           bytes  _producerSig) anonymous;

    event ConsumerTerminated(uint    indexed _event_code,
                             address indexed _consumer,
                             address indexed _producer,
                             bytes32 indexed _agreementID,
                             uint    _reason_code) anonymous;

    event ProducerTerminated(uint    indexed _event_code,
                              address indexed _consumer,
                              address indexed _producer,
                              bytes32 indexed _agreementID,
                              uint    _reason_code) anonymous;

    event TerminateFraudAlert(uint    indexed _event_code,
                              address indexed _consumer,
                              address indexed _producer,
                              bytes32 indexed _agreementID,
                              uint    _reason_code) anonymous;

    event AdminDeleted(uint    indexed _event_code,
                       address indexed _consumer,
                       address indexed _producer,
                       bytes32 indexed _agreementID,
                       uint    _reason_code) anonymous;

    // This function is invoked by the Consumer account to establish an agreemenmt in the blockchain
     function create_agreement(bytes32 _agreementID, bytes32 _contractHash, bytes _producerSig, address _counterParty) returns (uint ret) {
        // Validate the inputs. The producer's signature and hash are checked in the verifySig function.
        // The agreement ID can be any binary 32 byte value, including all zeroes.

        if (_counterParty == address(0)) {
            return 2;
        }

        // Make sure there isn't already an agreement in place. If so, don't do anything.
        if (callerIsConsumer(_counterParty, _agreementID) || callerIsProducer(_counterParty, _agreementID)) {
            return 3;
        }

        // Verify that the counter party did sign the contract hash. If so, we will save a record of the agreement
        // and emit events to indicate it's existence.
        if (verifySig(_contractHash, _producerSig, _counterParty) == true) {
            allAgreements[tx.origin].agreementInstances[_counterParty].theAgreement[_agreementID] = finalAgreement({contractHash: _contractHash,
                                                                                                                    agreementID:  _agreementID,
                                                                                                                    producerSig:  _producerSig});
            CreatedAgreement(uint(event_codes.created), tx.origin, _counterParty, _agreementID);
            CreatedDetail(uint(event_codes.created_detail), tx.origin, _counterParty, _agreementID, _contractHash, _producerSig);
            return 0;
        } else {
            CreateFraudAlert(uint(event_codes.create_fraud_alert), tx.origin, _counterParty, _agreementID, _contractHash, _producerSig);
            return 1;
        }
    }

    // This function is invoked by either the consumer or the producer to terminate an agreement.
    function terminate_agreement(address _otherParty, bytes32 _agreementID, uint _reason_code) returns (uint ret) {
        if (callerIsConsumer(_otherParty, _agreementID)) {
            delete allAgreements[tx.origin].agreementInstances[_otherParty].theAgreement[_agreementID];
            ConsumerTerminated(uint(event_codes.consumer_terminated), tx.origin, _otherParty, _agreementID, _reason_code);
            return 0;
        } else if (callerIsProducer(_otherParty, _agreementID)) {
            delete allAgreements[_otherParty].agreementInstances[tx.origin].theAgreement[_agreementID];
            ProducerTerminated(uint(event_codes.producer_terminated), _otherParty, tx.origin, _agreementID, _reason_code);
            return 0;
        }
        TerminateFraudAlert(uint(event_codes.terminate_fraud_alert), _otherParty, tx.origin, _agreementID, _reason_code);
        return 1;
    }

    // This function is invoked by either the consumer or the producer to retrieve the contract hash for an agreement
    // that exists between them.
    function get_contract_hash(address _otherParty, bytes32 _agreementID) constant returns (bytes32 _con) {
        if (callerIsConsumer(_otherParty, _agreementID)) {
            return allAgreements[tx.origin].agreementInstances[_otherParty].theAgreement[_agreementID].contractHash;
        } else if (callerIsProducer(_otherParty, _agreementID)) {
            return allAgreements[_otherParty].agreementInstances[tx.origin].theAgreement[_agreementID].contractHash;
        }
        return "";
    }

    // This function is invoked by either the consumer or the producer to retrieve the producer's signature for an agreement
    // that exists between them.
    function get_producer_signature(address _otherParty, bytes32 _agreementID) constant returns (bytes _sig) {
        if (callerIsConsumer(_otherParty, _agreementID)) {
            return allAgreements[tx.origin].agreementInstances[_otherParty].theAgreement[_agreementID].producerSig;
        } else if (callerIsProducer(_otherParty, _agreementID)) {
            return allAgreements[_otherParty].agreementInstances[tx.origin].theAgreement[_agreementID].producerSig;
        }
        return "";
    }

    // Internal function used to verify that a digital signature came from a specific ethereum address.
    // Returns true if the input hash and signature came from the input address, false otherwise.
    //
    function verifySig(bytes32 _contractHash, bytes _producerSig, address _counterParty) constant internal returns (bool ret) {
        // Use solidity ecrecover function to verify the producer's signature of the smarter contract. In order to do that
        // we need to decompose the signature into it's eliptic curve pieces and pass them into the builtin ecrecover
        // function.
        //

        // We expect _producerSig to have a length of 65 bytes
        if (_producerSig.length != 65)
            return false;

        // Pull the signature apart
        bytes32 r;
        bytes32 s;
        uint8 v;

        assembly {
            // The input signature is stored in memory in ABI format, which means it is preceded by a 32 byte word
            // indicating the length of the signature. First we need to skip over the length field to get to the first
            // 32 bytes of the signature.  The same thing is done to skip over the second 32 bytes of the signature.
            // The last byte is obtained by ANDing the last byte of the signature with x'FF'.
            r := mload(add(_producerSig, 32))
            s := mload(add(_producerSig, 64))
            v := and(mload(add(_producerSig, 65)), 255)
        }

        // toleration for old ethereum, we might not need this
        if (v < 27)
            v += 27;

        // Verify that the signature came from the counterparty
        if ( _counterParty == ecrecover(_contractHash, v, r, s) ) {
            return true;
        } else {
            return false;
        }
    }

    // Helper functions
    //
    // This function is used to determine if the caller is the playing the role of consumer in an
    // existing agreement with the counter party.
    function callerIsConsumer(address _otherParty, bytes32 _agreementID) constant returns (bool h) {
        var a = allAgreements[tx.origin].agreementInstances[_otherParty].theAgreement[_agreementID];
        if (a.producerSig.length != 0) {
            return true;
        }
        return false;
    }

    // This function is used to determine if the caller is the playing the role of producer in an
    // existing agreement with the counter party.
    function callerIsProducer(address _otherParty, bytes32 _agreementID) constant returns (bool h) {
        var a = allAgreements[_otherParty].agreementInstances[tx.origin].theAgreement[_agreementID];
        if (a.producerSig.length != 0) {
            return true;
        }
        return false;
    }

    // Administrative functions
    //
    // The account that owns this contract can delete anything in it.
    function admin_delete_agreement(address _consumer, address _producer,  bytes32 _agreementID, uint _reason_code) returns (uint ret) {
        if (tx.origin == owner) {
            var a = allAgreements[_consumer].agreementInstances[_producer].theAgreement[_agreementID];
            if (a.producerSig.length != 0) {
                delete allAgreements[_consumer].agreementInstances[_producer].theAgreement[_agreementID];
                AdminDeleted(uint(event_codes.admin_deleted), _consumer, _producer, _agreementID, _reason_code);
                return 0;
            } else {
                return 1;
            }
        } else {
            return 2;
        }
    }

    // Constructor, to establish who has admin control of this instance.
    function agreements() {
        owner = msg.sender;
    }

}
