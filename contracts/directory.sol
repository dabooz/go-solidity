contract directory {
    // The owner of the directory
    address owner;
    
    // A struct that holds the registered contract and the account of
    // the registerer. This is a leaf node in the directory.
    struct entry {
        address contract_addr;
        address contract_owner;
    }

    // A mapping of names to version number. This is the first level
    // of the directory.
    mapping (string => version) names;
    
    // A mapping of version to contract addresses. This is the second level
    // of the directory.
    struct version {
        mapping (uint => entry) versions;
    }

    // Events represent a history of eveything that happened in the directory.
    enum event_codes {
        add_entry_event_code,
        delete_entry_event_code
    }

    event AddEntry(uint indexed _eventcode, address indexed _adder, uint indexed version, address indexed _contract, string _name) anonymous;
    event DeleteEntry(uint indexed _eventcode, address indexed _deleter, uint indexed version, address indexed _contract, string _name) anonymous;

    // contructor
    function directory() {
        // Save owner address
        owner = msg.sender;
    }

    // Add an entry to the directory
    function add_entry(string _entry_name, address _address, uint _version) returns (uint r) {
        if (bytes(_entry_name).length == 0 || _address == address(0)) {
            return 1;
        }
        var addr = names[_entry_name].versions[_version].contract_addr;
        if (addr == address(0)) {
            names[_entry_name].versions[_version] = entry({contract_addr:_address,
                                        contract_owner:tx.origin});
            AddEntry(uint(event_codes.add_entry_event_code), tx.origin, _version, _address, _entry_name);
            return 0;
        } else {
            return 2;
        }
    }

    // Retrieve a specific entry from the directory, defaults to version zero.
    function get_entry(string _entry_name) constant returns (address r) {
        return names[_entry_name].versions[0].contract_addr;
    }

    // Retrieve a specific entry from the directory by version
    function get_entry_by_version(string _entry_name, uint _version) constant returns (address r) {
        return names[_entry_name].versions[_version].contract_addr;
    }

    // Remove a specific entry from the directory. Only the owner of the entry
    // can remove it.
    function delete_entry(string _entry_name, uint _version) returns (bool r) {
        var e = names[_entry_name].versions[_version];
        if (e.contract_owner == tx.origin && e.contract_addr != address(0)) {
            DeleteEntry(uint(event_codes.delete_entry_event_code), tx.origin, _version, e.contract_addr, _entry_name);
            delete names[_entry_name].versions[_version];
            return true;
        } else {
            return false;
        }
    }

    // Used to get rid of the contract
    function kill() {
        if (msg.sender == owner) suicide(owner);
    }
}
