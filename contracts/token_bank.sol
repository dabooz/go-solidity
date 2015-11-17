contract authorized_caller {
    function get_owner() constant returns (address r) {}
}

contract token_bank {

    // The minter can create Bacon for any entity.
    // The minter owns the Bacon contract instance.
    address minter;

    // This is the total amount of currency available in the ecosystem.
    uint total_currency;

    // Holds the account balance of all participants.
    mapping (address => uint) balances;

    // Holds the loan balances of all participants.
    mapping (address => uint) loans;

    uint constant loan_limit = 1000;

    // This represents the state of an escrow proposal
    struct proposal {
        bool proposer_vote;
        bool counter_party_vote;
        uint amount;
    }

    // This map ties the parties to a smart contract
    // key is the shared contract address
    struct shared_contract {
        mapping (address => proposal) proposals;
    }

    // This map ties the counter party to the proposer
    // key is the counter party
    struct counter_party {
        mapping (address => shared_contract) counter_parties;
    }

    // This map holds all info for any proposer
    // key is the proposer
    mapping (address => counter_party) escrow;

    // Events
    enum event_codes {
        mint,                           // tokens are being created
        loan_created,                   // a loan has been created
        loan_extended,                  // a loan was extended
        loan_repaid,                    // a loan was repaid
        transfer,                       // token transferred
        escrow_created,                 // escrow proposal created
        escrow_cancelled,               // potential agreement aborted
        escrow_counterparty_accepted,   // counterparty accepts escrow
        escrow_proposer_accepted,       // proposer accepts escrow
        escrow_closed                   // escrow has closed
    }
    event Mint(uint indexed _event_code, uint _value);
    event ObtainLoan(uint indexed _event_code, address _from, uint _value);
    event ExtendLoan(uint indexed _event_code, address _from, uint _value);
    event RepayLoan(uint indexed _event_code, address _from, uint _value);
    event Transfer(uint indexed _event_code, address _from, address _to, uint _value);
    event NewProposal(uint indexed _event_code, address _from, address _to, address indexed _contract);
    event CancelProposal(uint indexed _event_code, address _from, address _to, address indexed _contract);
    event CounterpartyAccepted(uint indexed _event_code, address _from, address _to, address indexed _contract);
    event ProposerVerified(uint indexed _event_code, address _from, address _to, address indexed _contract);
    event ProposalCompleted(uint indexed _event_code, address _from, address _to, address indexed _contract);

    uint escrow_debug;
    
    function get_debug() constant returns (uint r) {
        return escrow_debug;
    }
    // Constructor, runs when this contract is deployed (aka instantiated)
    function token_bank() {
        minter = msg.sender;
        total_currency = 1000000000000000;
        escrow_debug = 0;
    }

    // Send currency to another party
    function transfer(address _receiver, uint _amount) returns (bool r) {
        if (balances[tx.origin] < _amount) return false;
        balances[tx.origin] -= _amount;
        balances[_receiver] += _amount;
        Transfer(uint(event_codes.transfer), tx.origin, _receiver, _amount);
        return true;
    }

    // Debug method,used to get a balances
    function account_balance() constant returns (uint balance) {
        return balances[tx.origin];
    }

    // Returns true when the input address has sufficient funds.
    // It should be marked as internal when no longer needed for debug.
    function hasAmount(address _addr, uint _amount) internal constant returns (bool res) {
        return balances[_addr]>=_amount;
    }

    function obtain_loan(uint _amount) returns (bool r) {
        if (exceeds_loan_limits(_amount)) return false;
        var existing_loan = loans[tx.origin];
        if (existing_loan != 0) return false;
        loans[tx.origin] = _amount;
        total_currency -= _amount;
        balances[tx.origin] += _amount;
        ObtainLoan(uint(event_codes.loan_created), tx.origin, _amount);
        return true;
    }
    function increase_loan(uint _amount) returns (bool r) {
        if (exceeds_loan_limits(_amount)) return false;
        var existing_loan = loans[tx.origin];
        if (existing_loan == 0) return false;
        loans[tx.origin] += _amount;
        total_currency -= _amount;
        balances[tx.origin] += _amount;
        ExtendLoan(uint(event_codes.loan_extended), tx.origin, _amount);
        return true;
    }
    function repay_loan(uint _amount) returns (bool r) {
        if (balances[tx.origin] < _amount) return false;
        total_currency += _amount;
        loans[tx.origin] -= _amount;
        balances[tx.origin] -= _amount;
        RepayLoan(uint(event_codes.loan_repaid), tx.origin, _amount);
        return true;    
    }
    function loan_balance() constant returns (uint r) {
        return loans[tx.origin];
    }
    function exceeds_loan_limits(uint _amount) internal constant returns (bool r) {
        if (_amount > total_currency || _amount > loan_limit) {
            return true;
        }
        return false;
    }

    // Create a new escrow proposal only if
    // a) there isn't already a proposal between the parties involving the input smart contract
    // b) the proposal includes a non-zero amount of crytocurrency
    function create_escrow(address _cp, address _contract, uint _amount) returns (bool rv) {
        escrow_debug = 0;
        if (_amount > 0 && hasAmount(tx.origin,_amount)) {
            escrow_debug = 1;
            var prop = escrow[tx.origin].counter_parties[_cp].proposals[_contract];
            escrow_debug = 2;
            if (prop.amount == 0) {
                escrow_debug = 3;
                escrow[tx.origin].counter_parties[_cp].proposals[_contract] = proposal({proposer_vote:false,
                                                                                        counter_party_vote:false,
                                                                                        amount:_amount});
                balances[tx.origin] -= _amount;
                NewProposal(uint(event_codes.escrow_created), tx.origin, _cp, _contract);
                escrow_debug = 4;
                return true;
            } else {
                escrow_debug = 5;
                return false;
            }
        } else {
            escrow_debug = 6;
            return false;
        }
    }

    // Either party can cancel the escrow at any time
    function cancel_escrow(address _proposer, address _cp, address _contract) returns (bool rv) {
        var prop = escrow[_proposer].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0 && (tx.origin == _proposer || tx.origin == _cp)) {
            balances[_proposer] += escrow[_proposer].counter_parties[_cp].proposals[_contract].amount;
            clear_escrow(_proposer, _cp, _contract);
            CancelProposal(uint(event_codes.escrow_cancelled), _proposer, _cp, _contract);
            return true;
        }
    }

    // Both parties might want to look at the proposal. This is done in a way that only the involved parties
    // can see the amount.
    function get_escrow_amount(address _proposer, address _cp, address _contract) constant returns (uint rv) {
        if (tx.origin == _proposer || tx.origin == _cp) {
            var prop = escrow[_proposer].counter_parties[tx.origin].proposals[_contract];
            return prop.amount;
        } else {
            return 0;
        }
    }
    function get_counterparty_accepted(address _proposer, address _cp, address _contract) constant returns (bool rv) {
        var prop = escrow[_proposer].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0) {
            return prop.counter_party_vote;
        }
        return false;
    }
    function get_proposer_accepted(address _proposer, address _cp, address _contract) constant returns (bool rv) {
        var prop = escrow[_proposer].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0) {
            return prop.proposer_vote;
        }
        return false;
    }
    // When the proposer is satisfied that the proposal has been accepted, they vote to
    // release the escrowed funds. The proposer can also unvote by passing false for the vote.
    function proposer_vote(address _cp, address _contract, bool _vote) returns (bool rv) {
        var prop = escrow[tx.origin].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0) {
            prop.proposer_vote = _vote;
            close_escrow(tx.origin, _cp, _contract);
            ProposerVerified(uint(event_codes.escrow_proposer_accepted), tx.origin, _cp, _contract);
            return true;
        } else {
            return false;
        }
    }

    // When the counter party is satisfied with the proposal, they vote to accept
    // the proposal. The counterparty can also unvote by passing false for the vote.
    function counter_party_vote(address _proposer, address _contract, bool _vote) returns (bool rv) {
        var prop = escrow[_proposer].counter_parties[tx.origin].proposals[_contract];
        if (prop.amount != 0) {
            prop.counter_party_vote = _vote;
            close_escrow(_proposer, tx.origin, _contract);
            CounterpartyAccepted(uint(event_codes.escrow_counterparty_accepted), _proposer, tx.origin, _contract);
            return true;
        } else {
            return false;
        }
    }

    // Escrow is closed when both parties have agreed on the proposal. Funds are transferred
    // to the counter party.
    function close_escrow(address _proposer, address _cp, address _contract) internal {
        var prop = escrow[_proposer].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0 && prop.proposer_vote && prop.counter_party_vote) {
            balances[_cp] += escrow[_proposer].counter_parties[_cp].proposals[_contract].amount;
            ProposalCompleted(uint(event_codes.escrow_closed), _proposer, _cp, _contract);
            clear_escrow(_proposer, _cp, _contract);
        }
    }

    // Escrow proposal is cleared when the escrow is closed or cancelled.
    function clear_escrow(address _proposer, address _cp, address _contract) internal {
        delete escrow[_proposer].counter_parties[_cp].proposals[_contract];
    }

    // Only the contract owner or a contract owned by the same entity can
    // create currency
    function mint(uint _amount) {
        authorized_caller sender = authorized_caller(msg.sender);
        address sender_owner = sender.get_owner();
        if (tx.origin == minter || sender_owner == minter) {
            total_currency += _amount;
            Mint(uint(event_codes.mint), _amount);
        }
    }

    // Only the contract owner or a contract owned by the same entity can
    // check total currency
    function get_total_currency() constant returns (uint r) {
        authorized_caller sender = authorized_caller(msg.sender);
        address sender_owner = sender.get_owner();
        if (tx.origin == minter || sender_owner == minter) {
            return total_currency;
        }
        return 0;
    }

    function get_minter() constant returns (address r) {
        return minter;
    }
    // Used to get rid of the contract
    function kill() {
        if (msg.sender == minter) suicide(minter);
    }
}