contract token_bank {
    function create_escrow(address _cp, address _contract, uint _amount) returns (bool rv) {}
    function cancel_escrow(address _proposer, address _cp, address _contract) returns (bool rv) {}
    function clear_escrow(address _proposer, address _cp, address _contract) {}
}

contract container_executor {
    address owner;
    string whisper;
    string agreement;
    address container_provider;
    token_bank piggy_bank;

    uint constant new_container_event_code = 1;
    uint constant execution_complete_event_code = 2;
    uint constant container_rejected_event_code = 3;
    event NewContainer(uint indexed _eventcode, string _id, address indexed _self);
    event ExecutionComplete(uint indexed _eventcode, string _id, address indexed _self);
    event ContainerRejected(uint indexed _eventcode, string _id, address indexed _self);

    function new_container(string _whisperId, string _agreementId, uint _amount) returns (bool r) {
        if (_amount == 0) {
            return false;
        }
        if (!in_contract()) {
            whisper = _whisperId;
            agreement = _agreementId;
            container_provider = tx.origin;
            NewContainer(new_container_event_code,_agreementId,this);
            piggy_bank.create_escrow(owner,address(this),_amount);
            return true;
        } else {
            return false;
        }
    }
    function reject_container() returns (bool r) {
        if (tx.origin == owner) {
            piggy_bank.cancel_escrow(container_provider, tx.origin, this);
            ContainerRejected(container_rejected_event_code,agreement,this);
            whisper = "";
            container_provider = address(0);
            agreement = "";
            return true;
        }
    }
    function exec_complete() returns (bool r) {
        if (tx.origin == owner) {
            ExecutionComplete(execution_complete_event_code,agreement,this);
            piggy_bank.clear_escrow(container_provider, tx.origin, this);
            agreement = "";
            whisper = "";
            container_provider = address(0);
            return true;
        } else {
            return false;
        }
    }
    function in_contract() constant returns (bool r) {
        if (container_provider == address(0)) {
            return false;
        } else {
            return true;
        }
    }
    function get_agreement_id() constant returns (string r) {
        return agreement;
    }
    function get_container_provider() constant returns (address r) {
        return container_provider;
    }
    function get_whisper() constant returns (string r) {
        return whisper;
    }
    function container_executor() {
        owner = msg.sender;
    }
    function set_bank(address _bank) {
        if (owner == msg.sender) {
            piggy_bank = token_bank(_bank);
        }
    }
    function get_bank() constant returns (address r) {
        return piggy_bank;
    }
    function get_owner() constant returns (address r) {
        return owner;
    }
    // Used to get rid of the contract
    function kill() {
        if (msg.sender == owner) suicide(owner);
    }
}

