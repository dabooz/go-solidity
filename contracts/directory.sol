contract directory {
    // The owner of the directory
    address owner;
    
    // A struct that holds the registered contract and the account of
    // the registerer.
    struct entry {
        address contract_addr;
        address contract_owner;
    }

    // A mapping of names to contract addresses. This is the heart
    // of the directory.
    mapping (bytes32 => entry) names;
    
    // A list of all registered names that behaves like an array.
    // The last_index variable holds the index (base zero) of the next
    // available slot in the array.
    mapping (uint => bytes32) index;
    uint last_index = 0;
    
    // This is working storage used to return the list of registered names.
    bytes32[] rr;

    // contructor
    function directory() {
        // Save owner address
        owner = msg.sender;
    }

    // Add an entry to the directory
    function add_entry(bytes32 _entry_name, address _address) returns (uint r) {
        if (_entry_name == 0 || _address == address(0)) {
            return 1;
        }
        var addr = names[_entry_name].contract_addr;
        if (addr == address(0)) {
            names[_entry_name] = entry({contract_addr:_address,
                                        contract_owner:tx.origin});
            index[last_index] = _entry_name;
            last_index += 1;
            return 0;
        } else {
            return 2;
        }
    }

    // Retrieve a specific entry from the directory
    function get_entry(bytes32 _entry_name) constant returns (address r) {
        return names[_entry_name].contract_addr;
    }

    // Retrieve the owner of a given entry. Only the owner of this contract
    // is allowed to retrieve this information.
    function get_entry_owner(bytes32 _entry_name) constant returns (address r) {
        if (tx.origin == owner) {
            return names[_entry_name].contract_owner;
        } else {
            return address(0);
        }
    }

    // Remove a specific entry from the directory. Only the owner of the entry
    // can remove it. Further, this logic is tricky because it also adjusts the
    // index list to ensure there are no gaps in it. This means that
    // registered names at a given index location may move over time
    // if entries are deleted.
    function delete_entry(bytes32 _entry_name) returns (bool r) {
        var e = names[_entry_name];
        if (e.contract_owner == tx.origin && e.contract_addr != address(0)) {
            delete names[_entry_name];
            uint i = 0;
            bool removed = false;
            while (i < last_index) {
                if (index[i] == _entry_name) {
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
    function get_names(uint _start, uint _end) constant returns (bytes32[] r) {
        rr.length = _end-_start+1;
        uint i = 0;
        while (i < _end-_start+1) {
            rr[i] = index[i+_start];
            i += 1;
        }
        return rr;
    }

    // Used to get rid of the contract
    function kill() {
        if (msg.sender == owner) suicide(owner);
    }
} 