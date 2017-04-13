contract metering {

    // The owner of this ethereum contract instance
    address owner;

    // This represents the final metering outcome of an agreement
    struct finalMeter {
        uint    count;                // The number of tokens earned
        uint    time;                 // The time in seconds when the metering occurred
        bytes32 agreementID;          // The agreement id used to identify the agreement
        bytes32 meterHash;            // SHA3 FIPS-202 hash of count, time and agreement id
        bytes   consumerMeterSig;     // The consumer's signature of the meter
        bytes32 contractHash;         // The SHA3 FIPS-202 hash of the agreed to merged policy document
        bytes   producerAgreementSig; // The producer's signature of the contract hash
        bytes   consumerAgreementSig; // The consumer's signature of the contract hash
    }

    // This map ties the parties to a specific agreement ID.
    // The key is the agreement ID.
    struct meteredOutcome {
        mapping (bytes32 => finalMeter) meterReading;
    }

    // This map ties the counter party to the producer.
    // The key is the counter party address (usually the consumer).
    struct counter_party {
        mapping (address => meteredOutcome) meters;
    }

    // This map holds all info for any producer.
    // The key is the producer's account address
    mapping (address => counter_party) allMetering;

    // Events are emitted from this smart contract when important things happen. The event code
    // indicates which event occurred and the data associated with the event provides the context.
    enum event_codes {
        created,                           // A final meter was created
        created_detail,                    // The details of a meter was created
        create_fraud_alert,                // The caller tried to create a fraudulent metering record
        admin_deleted                      // The ethereum smart contract owner deleted the metering record
    }

    event CreatedMeter(uint    indexed _event_code,
                       address indexed _producer,
                       address indexed _consumer,
                       bytes32 indexed _agreementID) anonymous;

    event CreatedMeterDetail(uint    indexed _event_code,
                             address indexed _producer,
                             address indexed _consumer,
                             bytes32 indexed _agreementID,
                             uint _count,
                             uint _time,
                             bytes32 _meterHash,
                             bytes _consumerMeterSig,
                             bytes32 _contractHash,
                             bytes _producerSig,
                             bytes _consumerSig) anonymous;

    event CreateFraudAlert(uint    indexed _event_code,
                           address indexed _producer,
                           address indexed _consumer,
                           bytes32 indexed _agreementID,
                           uint _count,
                           uint _time,
                           bytes32 _meterHash,
                           bytes _consumerMeterSig,
                           bytes32 _contractHash,
                           bytes _producerSig,
                           bytes _consumerSig) anonymous;

    event AdminDeleted(uint    indexed _event_code,
                       address indexed _producer,
                       address indexed _consumer,
                       bytes32 indexed _agreementID) anonymous;

    // This function is invoked by the Producer account to establish a final meter in the blockchain. It is the
    // producer's responsibility to submit this transaction when it wants to ensure that it's meter reading is
    // recorded. It is the consumer's responsibility to authorize the count and time. This authorization is
    // obtained by the producer when the consumer signs the meter reading. This contract will verify that the
    // consumer signed it before writing the record into the blockchain. Both parties must also sign
    // the hash of the agreement over which this meter reading is being used. The agreement includes the terms
    // of the metering, so it's important for both parties to agree to the terms and for their agreement to also
    // be recorded in the metering record.
    function create_meter(uint _amount,
                          uint _time,
                          bytes32 _agreementID,
                          bytes32 _meterHash,
                          bytes _consumerMeterSig,
                          bytes32 _contractHash,
                          bytes _producerSig,
                          bytes _consumerSig,
                          address _consumerAddress) returns (uint ret) {

        // Validate the inputs. The signatures and hash are checked in the verifySig function.
        // The agreement ID can be any binary 32 byte value, including all zeroes.
        if (_consumerAddress == address(0) || _amount == 0 || _time == 0) {
            return 2;
        }

        // It is an error if a meter record already exists with a later time or a larger count value.
        // Valid metered values can only go up.
        var mr = getReading(_consumerAddress, _agreementID);
        if (mr.count != 0 && (mr.count > _amount || mr.time > _time)) {
            return 3;
        }

        // Verify that all the signatures check out ok.
        if (verifySig(_contractHash, _consumerSig, _consumerAddress) == true &&
            verifySig(_contractHash, _producerSig, tx.origin) == true &&
            verifySig(_meterHash, _consumerMeterSig, _consumerAddress) == true) {
            allMetering[tx.origin].meters[_consumerAddress].meterReading[_agreementID] = finalMeter({
                count:                _amount,
                time:                 _time,
                agreementID:          _agreementID,
                meterHash:            _meterHash,
                consumerMeterSig:     _consumerMeterSig,
                contractHash:         _contractHash,
                producerAgreementSig: _producerSig,
                consumerAgreementSig: _consumerSig
                });

            // Emit events to indicate the creation of the meter reading on the blockchain.
            CreatedMeter(uint(event_codes.created), tx.origin, _consumerAddress, _agreementID);
            CreatedMeterDetail(uint(event_codes.created_detail), tx.origin, _consumerAddress, _agreementID, _amount, _time, _meterHash, _consumerMeterSig, _contractHash, _producerSig, _consumerSig);
            return 0;
        } else {
            CreateFraudAlert(uint(event_codes.create_fraud_alert), tx.origin, _consumerAddress, _agreementID, _amount, _time, _meterHash, _consumerMeterSig, _contractHash, _producerSig, _consumerSig);
            return 1;
        }
    }

    // This function retrieves a specific meter reading, returning the metered count and the time of meter record.
    function read_meter(bytes32 _agreementID, address _counterParty) constant returns (uint count, uint time) {
        if (callerIsConsumer(_counterParty, _agreementID)) {
            var r = allMetering[_counterParty].meters[tx.origin].meterReading[_agreementID];
            return (r.count, r.time);
        } else if (callerIsProducer(_counterParty, _agreementID)) {
            var s = allMetering[tx.origin].meters[_counterParty].meterReading[_agreementID];
            return (s.count, s.time);
        }
    }

    // Internal function used to verify that a digital signature came from a specific ethereum address.
    // Returns true if the input hash and signature came from the input address, false otherwise.
    //
    function verifySig(bytes32 _contractHash, bytes _sig, address _counterParty) constant internal returns (bool ret) {
        // Use solidity ecrecover function to verify the input signature. In order to do that we need to decompose
        // the signature into it's eliptic curve pieces and pass them into the builtin ecrecover function.
        //

        // We expect _sig to have a length of 65 bytes
        if (_sig.length != 65)
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
            r := mload(add(_sig, 32))
            s := mload(add(_sig, 64))
            v := and(mload(add(_sig, 65)), 255)
        }

        // toleration for old ethereum, we might not need this
        if (v < 27)
            v += 27;

        // Verify that the signature came from the counterparty
        if ( _counterParty == ecrecover(sha3("\x19Ethereum Signed Message:\n32",_contractHash), v, r, s) ) {
            return true;
        } else {
            return false;
        }
    }

    // Helper functions
    //
    // This function is used to determine if the caller is playing the role of consumer in an
    // existing meter with the counter party.
    function callerIsConsumer(address _otherParty, bytes32 _agreementID) constant internal returns (bool h) {
        var a = allMetering[_otherParty].meters[tx.origin].meterReading[_agreementID];
        if (a.producerAgreementSig.length != 0) {
            return true;
        }
        return false;
    }

    // This function is used to determine if the caller is playing the role of producer in an
    // existing meter with the counter party.
    function callerIsProducer(address _otherParty, bytes32 _agreementID) constant internal returns (bool h) {
        var a = allMetering[tx.origin].meters[_otherParty].meterReading[_agreementID];
        if (a.producerAgreementSig.length != 0) {
            return true;
        }
        return false;
    }

    // This function returns a meter reading.
    function getReading(address _otherParty, bytes32 _agreementID) constant internal returns (finalMeter _fm) {
        return allMetering[tx.origin].meters[_otherParty].meterReading[_agreementID];
    }

    // Administrative functions
    //
    // The account that owns this contract can delete anything in it.
    function admin_delete_meter(address _producer, address _consumer,  bytes32 _agreementID) returns (uint ret) {
        if (tx.origin == owner) {
            var a = allMetering[_producer].meters[_consumer].meterReading[_agreementID];
            if (a.producerAgreementSig.length != 0) {
                delete allMetering[_producer].meters[_consumer].meterReading[_agreementID];
                AdminDeleted(uint(event_codes.admin_deleted), _producer, _consumer, _agreementID);
                return 0;
            } else {
                return 1;
            }
        } else {
            return 2;
        }
    }

    // Constructor, to establish who has admin control of this instance.
    function metering() {
        owner = msg.sender;
    }

}