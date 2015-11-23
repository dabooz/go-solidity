contract whisper_directory {
    // The owner of the directory
    address owner;

    // A mapping of ethereum account to valid whisper account. This is the heart
    // of the directory.
    mapping (address => string) whisper_accounts;

    // contructor
    function whisper_directory() {
        // Save owner address
        owner = msg.sender;
    }

    // Add an entry to the directory
    function add_entry(string _whisper_account) returns (bool r) {
        whisper_accounts[tx.origin] = _whisper_account;
        return true;
    }

    // Retrieve a specific entry from the directory
    function get_entry(address _account) constant returns (string r) {
        return whisper_accounts[_account];
    }

    // Remove a specific entry from the directory.
    function delete_entry() returns (bool r) {
        delete whisper_accounts[tx.origin];
        return true;
    }
    
    // Used to get rid of the contract
    function kill() {
        if (msg.sender == owner) suicide(owner);
    }
}
