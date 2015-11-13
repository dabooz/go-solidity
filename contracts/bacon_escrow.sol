contract authorized_caller {
    function get_owner() constant returns (address r) {}
}

// The bacon_escrow contract is a singleton.
// We would expect one instance in the block chain, and we would expect
// all ecosystem participants to use the same Bacon contract instance,
// otherwise the economy would be fragmented.
contract bacon_escrow {

    // The minter can create Bacon for any entity.
    // The minter owns the Bacon contract instance.
    address minter;

    // Holds the account balance of all participants.
    mapping (address => uint) balances;

    // These are events that are emitted from this contract
    uint mint_event_code = 1;
    uint transfer_event_code = 2;
    uint proposal_created_event_code = 3;
    uint proposal_cancelled_event_code = 4;
    uint counterparty_accepted_event_code = 5;
    uint proposer_verified_event_code = 6;
    uint proposal_completed_event_code = 7;

    event Mint(uint indexed _event_code, address _to, uint _value);
    event Transfer(uint indexed _event_code, address _from, address _to, uint _value);
    event NewProposal(uint indexed _event_code, address _from, address _to, address indexed _contract);
    event CancelProposal(uint indexed _event_code, address _from, address _to, address indexed _contract);
    event CounterpartyAccepted(uint indexed _event_code, address _from, address _to, address indexed _contract);
    event ProposerVerified(uint indexed _event_code, address _from, address _to, address indexed _contract);
    event ProposalCompleted(uint indexed _event_code, address _from, address _to, address indexed _contract);

    // These structs and mappings represent a 2 party relationship where
    // a proposer makes a proposal to a counterparty. The proposal is related
    // to a specific smart contract (instance) and includes an amount of
    // crytocurrency that will be exchanged when both parties agree (vote) to
    // accept the proposal. The proposer votes to accept the proposal when it
    // is satisfied that the counterparty has enacted their part of the
    // relationship (including accepting the proposal). This may happen using
    // an out of band protocol.

    // This represents the state of the proposal
    struct proposal {
        uint proposer_vote;
        uint counter_party_vote;
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

    // Create a new escrow proposal only if
    // a) there isn't already a proposal between the parties involving the input smart contract
    // b) the proposal includes a non-zero amount of crytocurrency
    function create_escrow(address _cp, address _contract, uint _amount) returns (bool rv) {
        if (_amount > 0 && hasAmount(tx.origin,_amount)) {
            var prop = escrow[tx.origin].counter_parties[_cp].proposals[_contract];
            if (prop.amount == 0) {
                escrow[tx.origin].counter_parties[_cp].proposals[_contract] = proposal({proposer_vote:0,counter_party_vote:0,amount:_amount});
                balances[tx.origin] -= _amount;
                NewProposal(proposal_created_event_code, tx.origin, _cp, _contract);
                return true;
            } else {
                return false;
            }
        } else {
            return false;
        }
    }

    // The proposer can cancel the proposal at any time for a refund of the escrowed funds
    function cancel_escrow(address _cp, address _contract) returns (bool rv) {
        balances[tx.origin] += escrow[tx.origin].counter_parties[_cp].proposals[_contract].amount;
        clear_escrow(tx.origin, _cp, _contract);
        CancelProposal(proposal_cancelled_event_code, tx.origin, _cp, _contract);
        return true;
    }

    // The counter party might want to look at the proposal. This is done in a way that no one other
    // than the counter party can see the proposal.
    function get_proposal_amount(address _proposer, address _contract) constant returns (uint rv) {
        var prop = escrow[_proposer].counter_parties[tx.origin].proposals[_contract];
        return prop.amount;
    }

    // When the proposer is satisfied that the proposal has been accepted, they vote to
    // release the escrowed funds.
    function proposer_vote(address _cp, address _contract, uint _vote) returns (bool rv) {
        var prop = escrow[tx.origin].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0 && _vote > 0) {
            prop.proposer_vote = _vote;
            close_escrow(tx.origin, _cp, _contract);
            ProposerVerified(proposer_verified_event_code, tx.origin, _cp, _contract);
            return true;
        } else {
            return false;
        }
    }

    // When the counter party is satisfied with the proposal, they vote to accept
    // the proposal.
    function counter_party_vote(address _proposer, address _contract, uint _vote) returns (bool rv) {
        var prop = escrow[_proposer].counter_parties[tx.origin].proposals[_contract];
        if (prop.amount != 0 && _vote > 0) {
            prop.counter_party_vote = _vote;
            close_escrow(_proposer, tx.origin, _contract);
            CounterpartyAccepted(counterparty_accepted_event_code, _proposer, tx.origin, _contract);
            return true;
        } else {
            return false;
        }
    }

    // Escrow is closed when both parties have agreed on the proposal. Funds are transferred
    // to the counter party.
    function close_escrow(address _proposer, address _cp, address _contract) internal {
        var prop = escrow[_proposer].counter_parties[_cp].proposals[_contract];
        if (prop.amount != 0 && prop.proposer_vote > 0 && prop.counter_party_vote > 0) {
            balances[_cp] += escrow[_proposer].counter_parties[_cp].proposals[_contract].amount;
            ProposalCompleted(proposal_completed_event_code, _proposer, _cp, _contract);
            clear_escrow(_proposer, _cp, _contract);
        }
    }

    // Escrow proposal is cleared when the escrow is closed or cancelled.
    function clear_escrow(address _proposer, address _cp, address _contract) internal {
        escrow[_proposer].counter_parties[_cp].proposals[_contract]=proposal({proposer_vote:0,counter_party_vote:0,amount:0});
    }

    // Constructor, runs when this contract is deployed (aka instantiated)
    function bacon_escrow() {
        minter = msg.sender;
    }

    // Only the contract owner or a contract owned by the same entity can create currency
    function mint(address _owner, uint _amount) {
        authorized_caller sender = authorized_caller(msg.sender);
        address sender_owner = sender.get_owner();
        if (tx.origin == minter || sender_owner == minter) {
            balances[_owner] += _amount;
            Mint(mint_event_code, _owner, _amount);
        }
    }

    // Send currency to another party
    function send(address _receiver, uint _amount) {
        if (balances[tx.origin] < _amount) return;
        balances[tx.origin] -= _amount;
        balances[_receiver] += _amount;
        Transfer(transfer_event_code, tx.origin, _receiver, _amount);
    }

    // Debug method,used to get a balances
    function queryBalance(address _addr) constant returns (uint balance) {
        return balances[_addr];
    }

    // Returns true when the input address has sufficient funds.
    // It should be marked as internal when no longer needed for debug.
    function hasAmount(address _addr, uint _amount) constant returns (bool res) {
        return balances[_addr]>=_amount;
    }
    function get_minter() constant returns (address r) {
        return minter;
    }

    // Used to get rid of the contract
    function kill() {
        if (msg.sender == minter) suicide(minter);
    }
}