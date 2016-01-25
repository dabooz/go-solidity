contract directory {
    // The owner of the directory
    address owner;
    
    // A struct that holds the registered contract and the account of
    // the registerer.
    struct entry {
        address contract_addr;
        address contract_owner;
    }

    // A mapping of names to version number. This is the heart
    // of the directory.
    mapping (bytes32 => version) names;
    
    // A mapping of version to contract addresses. This is the heart
    // of the directory.
    struct version {
        mapping (uint => entry) versions;
    }
    
    // A list of all registered names that behaves like an array.
    // The last_index variable holds the index (base zero) of the next
    // available slot in the array.
    struct name_entry {
        bytes32 name;
        uint version;
    }
    mapping (uint => name_entry) index;
    uint last_index = 0;
    
    // This is working storage used to return the list of registered names.
    bytes32[] rr;

    // Events represent a history of eveything that happened in the directory.
    enum event_codes {
        add_entry_event_code,
        delete_entry_event_code
    }

    event AddEntry(uint indexed _eventcode, address indexed _adder, uint indexed version, address indexed _contract, bytes32 _name) anonymous;
    event DeleteEntry(uint indexed _eventcode, address indexed _deleter, uint indexed version, address indexed _contract, bytes32 _name) anonymous;

    // contructor
    function directory() {
        // Save owner address
        owner = msg.sender;
    }

    // Add an entry to the directory
    function add_entry(bytes32 _entry_name, address _address, uint _version) returns (uint r) {
        if (_entry_name == 0 || _address == address(0)) {
            return 1;
        }
        var addr = names[_entry_name].versions[_version].contract_addr;
        if (addr == address(0)) {
            names[_entry_name].versions[_version] = entry({contract_addr:_address,
                                        contract_owner:tx.origin});
            AddEntry(uint(event_codes.add_entry_event_code), tx.origin, _version, _address, _entry_name);
            index[last_index] = name_entry({name:_entry_name,
                                            version:_version});
            last_index += 1;
            return 0;
        } else {
            return 2;
        }
    }

    // Retrieve a specific entry from the directory
    function get_entry(bytes32 _entry_name) constant returns (address r) {
        return names[_entry_name].versions[0].contract_addr;
    }

    // Retrieve a specific entry from the directory by version
    function get_entry_by_version(bytes32 _entry_name, uint _version) constant returns (address r) {
        return names[_entry_name].versions[_version].contract_addr;
    }

    // Retrieve the owner of a given entry. Only the owner of this contract
    // is allowed to retrieve this information.
    function get_entry_owner(bytes32 _entry_name, uint _version) constant returns (address r) {
        if (tx.origin == owner) {
            return names[_entry_name].versions[_version].contract_owner;
        } else {
            return address(0);
        }
    }

    // Remove a specific entry from the directory. Only the owner of the entry
    // can remove it. Further, this logic is tricky because it also adjusts the
    // index list to ensure there are no gaps in it. This means that
    // registered names at a given index location may move over time
    // if entries are deleted.
    function delete_entry(bytes32 _entry_name, uint _version) returns (bool r) {
        var e = names[_entry_name].versions[_version];
        if (e.contract_owner == tx.origin && e.contract_addr != address(0)) {
            DeleteEntry(uint(event_codes.delete_entry_event_code), tx.origin, _version, e.contract_addr, _entry_name);
            delete names[_entry_name].versions[_version];
            uint i = 0;
            bool removed = false;
            while (i < last_index) {
                if (index[i].name == _entry_name && index[i].version == _version) {
                    delete index[i];
                    removed = true;
                } else if (removed == true) {
                    index[i-1] = index[i];
                }
                i += 1;
            }
            delete index[last_index-1];
            last_index -= 1;
            return true;
        } else {
            return false;
        }
    }

    // Get some or all of the names registered in the directory
    // If a name is registered under multiple versions it will appear
    // multiple times in the output array.
    function get_names(uint _start, uint _end) constant returns (bytes32[] r) {
        uint i = _start;
        if (_start >= last_index) {
            i = last_index;
        }
        uint stop = _end+1;
        if (_end >= last_index) {
            stop = last_index;
        }
        rr.length = stop-i;
        while (i < stop) {
            rr[i] = index[i].name;
            i += 1;
        }
        return rr;
    }

    // Used to get rid of the contract
    function kill() {
        if (msg.sender == owner) suicide(owner);
    }
}
