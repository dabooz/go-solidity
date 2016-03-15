contract authorized_caller {
    function get_owner() constant returns (address r) {}
}

contract container_executor {
    function cancel_container(uint _amount) returns (bool r) {}
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

    uint constant loan_limit = 100000000;

    // This represents the state of an escrow proposal
    struct proposal {
        bool proposer_vote;
        bool counter_party_vote;
        bool cancelled;
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
        escrow_proposer_paid,           // proposer paid device owner
        escrow_refunded                 // escrow refunded to proposer
    }

    event Mint(uint indexed _event_code, address indexed owner, address indexed _from, uint indexed _value) anonymous;
    event ObtainLoan(uint indexed _event_code, address indexed _from, uint indexed _value) anonymous;
    event ExtendLoan(uint indexed _event_code, address indexed _from, uint indexed _value) anonymous;
    event RepayLoan(uint indexed _event_code, address indexed _from, uint indexed _value) anonymous;
    event Transfer(uint indexed _event_code, address indexed _from, address indexed _to, uint indexed _value) anonymous;
    event NewProposal(uint indexed _event_code, address indexed _from, address indexed _to, address indexed _contract, uint _amount) anonymous;
    event CancelProposal(uint indexed _event_code, address indexed _from, address indexed _to, address indexed _contract) anonymous;
    event CounterpartyAccepted(uint indexed _event_code, address indexed _from, address indexed _to, address indexed _contract) anonymous;
    event ProposerVerified(uint indexed _event_code, address indexed _from, address indexed _to, address indexed _contract) anonymous;
    event ProposerPaid(uint indexed _event_code, address indexed _from, address indexed _to, address indexed _contract, uint _value) anonymous;
    event EscrowRefund(uint indexed _event_code, address indexed _from, address indexed _to, address indexed _contract, uint _value) anonymous;

    // Constructor, runs when this contract is deployed (aka instantiated)
    function token_bank() {
        minter = msg.sender;
        total_currency = 1000000000000000000000;
    }

    // Send currency to another party
    function transfer(address _receiver, uint _amount) returns (bool r) {
        if (balances[tx.origin] < _amount) return false;
        balances[tx.origin] -= _amount;
        balances[_receiver] += _amount;
        Transfer(uint(event_codes.transfer), tx.origin, _receiver, _amount);
        return true;
    }

    // Get a balance
    function account_balance() constant returns (uint balance) {
        return balances[tx.origin];
    }
    function account_balance_by_addr(address _addr) constant returns (uint balance) {
        if (tx.origin == minter || tx.origin == _addr) {
            return balances[_addr];
        } else {
            return 0;
        }
    }

    // Returns true when the input address has sufficient funds.
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
    function loan_balance_by_addr(address _addr) constant returns (uint balance) {
        if (tx.origin == minter || tx.origin == _addr) {
            return loans[_addr];
        } else {
            return 0;
        }
    }
    function exceeds_loan_limits(uint _amount) internal constant returns (bool r) {
        if (_amount > total_currency || _amount > loan_limit) {
            return true;
        }
        return false;
    }

    // Create a new escrow proposal only if
    // a) there isn't already a proposal between the parties involving the input smart contract
    // b) the proposal includes a non-zero number of tokens
    function create_escrow(address _cp, address _contract, uint _amount) returns (bool rv) {
        if (_amount > 0 && hasAmount(tx.origin,_amount)) {
            var prop = escrow[tx.origin].counter_parties[_cp].proposals[_contract];
            if (prop.amount == 0) {
                escrow[tx.origin].counter_parties[_cp].proposals[_contract] = proposal({proposer_vote:false,
                                                                                        counter_party_vote:false,
                                                                                        cancelled:false,
                                                                                        amount:_amount});
                balances[tx.origin] -= _amount;
                NewProposal(uint(event_codes.escrow_created), tx.origin, _cp, _contract, _amount);
                return true;
            } else {
                return false;
            }
        } else {
            return false;
        }
    }

    // When the counter party is satisfied with the proposal, they vote to accept
    // the proposal.
    function counter_party_vote(address _proposer, address _contract, bool _vote) returns (bool rv) {
        var prop = escrow[_proposer].counter_parties[tx.origin].proposals[_contract];
        if (prop.amount != 0) {
            prop.counter_party_vote = _vote;
            CounterpartyAccepted(uint(event_codes.escrow_counterparty_accepted), _proposer, tx.origin, _contract);
            return true;
        } else {
            return false;
        }
    }

    // When the proposer is satisfied that the proposal has been accepted, the agreement
    // is complete.
    function proposer_vote(address _cp, address _contract, bool _vote) returns (bool rv) {
        var prop = escrow[tx.origin].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0) {
            prop.proposer_vote = _vote;
            ProposerVerified(uint(event_codes.escrow_proposer_accepted), tx.origin, _cp, _contract);
            return true;
        } else {
            return false;
        }
    }

    // When the proposer is ready to pay the counter party, they use this function to transfer
    // funds and refill the escrow. The proposer will never pay more than what is escrowed.
    // After the device owner is paid, the proposer must have funds to refill the escrow amount
    // for the next segment of time. If the proposer doesn't have sufficient funds, then the
    // agreement will be cancelled.
    function make_payment(address _cp, address _contract, uint _amount) returns (bool rv) {
        var prop = escrow[tx.origin].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0 && prop.cancelled == false && prop.proposer_vote == true && prop.counter_party_vote == true) {
            var pay = _amount;
            if (_amount > prop.amount) {
                pay = prop.amount;
            }
            balances[_cp] += pay;
            ProposerPaid(uint(event_codes.escrow_proposer_paid), tx.origin, _cp, _contract, _amount);
            var remains = prop.amount - pay;
            if (hasAmount(tx.origin, pay)) {
                balances[tx.origin] -= pay;
            } else {
                balances[tx.origin] += remains;
                var device_contract = container_executor(_contract);
                device_contract.cancel_container(pay);
            }
            return true;
        } else {
            return false;
        }
    }

    // Either party can cancel the escrow at any time. This method should only be called by the
    // container_executor contract.
    function cancel_escrow(address _proposer, address _cp, address _contract, uint _amount) returns (bool rv) {
        if (tx.origin != _proposer && tx.origin != _cp) return false;
        var prop = escrow[_proposer].counter_parties[_cp].proposals[_contract];
        if (tx.origin == _proposer) {
            if (prop.amount != 0 && (prop.proposer_vote == false || prop.counter_party_vote == false)) {
                balances[_proposer] += prop.amount;
                CancelProposal(uint(event_codes.escrow_cancelled), _proposer, _cp, _contract);
                EscrowRefund(uint(event_codes.escrow_refunded), _proposer, _cp, _contract, prop.amount);
                delete escrow[_proposer].counter_parties[_cp].proposals[_contract];
            } else {
                if (prop.amount != 0 && _amount > 0) {
                    var pay = _amount;
                    if (_amount > prop.amount) {
                        pay = prop.amount;
                    }
                    balances[_cp] += pay;
                    ProposerPaid(uint(event_codes.escrow_proposer_paid), _proposer, _cp, _contract, pay);
                    CancelProposal(uint(event_codes.escrow_cancelled), _proposer, _cp, _contract);
                    var remains = prop.amount - pay;
                    balances[tx.origin] += remains;
                    EscrowRefund(uint(event_codes.escrow_refunded), _proposer, _cp, _contract, remains);
                    delete escrow[_proposer].counter_parties[_cp].proposals[_contract];
                } else {
                    return false;
                }
            }
        } else {
            if (tx.origin == _cp) {
                prop.cancelled = true;
                if (prop.amount != 0 && (prop.proposer_vote == false || prop.counter_party_vote == false)) {
                    balances[_proposer] += prop.amount;
                    EscrowRefund(uint(event_codes.escrow_refunded), _proposer, _cp, _contract, prop.amount);
                    CancelProposal(uint(event_codes.escrow_cancelled), _proposer, _cp, _contract);
                    delete escrow[_proposer].counter_parties[_cp].proposals[_contract];
                }
            }
        }
        return true;
    }

    // Both parties might want to look at the proposal. This is done in a way that only the involved parties
    // can see the amount (or the system admin).
    function get_escrow_amount(address _proposer, address _cp, address _contract) constant returns (uint rv) {
        if (tx.origin == _proposer || tx.origin == _cp || tx.origin == minter) {
            var prop = escrow[_proposer].counter_parties[_cp].proposals[_contract];
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
    function get_agreement_cancelled(address _proposer, address _cp, address _contract) constant returns (bool rv) {
        var prop = escrow[_proposer].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0) {
            return prop.cancelled;
        }
        return false;
    }

    // Only the contract owner or a contract owned by the same entity can
    // create currency
    function mint(uint _amount) {
        authorized_caller sender = authorized_caller(msg.sender);
        address sender_owner = sender.get_owner();
        if (tx.origin == minter || sender_owner == minter) {
            total_currency += _amount;
            Mint(uint(event_codes.mint), minter, sender_owner, _amount);
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
