contract whisper_directory {
    // The owner of the directory
    address owner;

    // A mapping of ethereum account to valid whisper account. This is the heart
    // of the directory.
    mapping (address => string) whisper_accounts;

    // Events represent a history of eveything that happened in the directory.
    enum event_codes {
        add_entry_event_code,
        delete_entry_event_code
    }

    event AddEntry(uint indexed _eventcode, address indexed _adder, string _whisper_account) anonymous;
    event DeleteEntry(uint indexed _eventcode, address indexed _deleter, string _whisper_account) anonymous;

    // contructor
    function whisper_directory() {
        // Save owner address
        owner = msg.sender;
    }

    // Add an entry to the directory. Only the caller's entry in the directory
    // is updated.
    function add_entry(string _whisper_account) returns (bool r) {
        whisper_accounts[tx.origin] = _whisper_account;
        AddEntry(uint(event_codes.add_entry_event_code), tx.origin, _whisper_account);
        return true;
    }

    // Retrieve a specific entry from the directory. Anyone can retrieve the
    // whisper account for a given ethereum account.
    function get_entry(address _account) constant returns (string r) {
        return whisper_accounts[_account];
    }

    // Remove a specific entry from the directory. The caller can only remove
    // his own entries.
    function delete_entry() returns (bool r) {
        var deleted_wa = whisper_accounts[tx.origin];
        DeleteEntry(uint(event_codes.delete_entry_event_code), tx.origin, deleted_wa);
        delete whisper_accounts[tx.origin];
        return true;
    }
    
    // Used to get rid of the contract
    function kill() {
        if (msg.sender == owner) suicide(owner);
    }
}
