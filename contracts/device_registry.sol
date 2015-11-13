contract container_executor {
    function get_owner() constant returns (address r) {}
}
contract token_bank {
    function mint(uint _amount) {}
}
contract device_registry {
    address owner;
    token_bank piggy_bank;
    mapping (address => Description) devices;
    struct Description {
        mapping (bytes32 => bytes32) values;
        mapping (uint => bytes32) index;
        uint last_index;
    }
    mapping (uint => address) index;
    uint last_index = 0;
    mapping (address => MyDevices) owned;
    struct MyDevices {
        mapping (uint => address) device;
        uint last_index;
    }
    bytes32[] desc_out;
    address[] search_out;
    function get_number_registered() constant returns (uint r) {
        return last_index;
    }
    function register(address _device, bytes32[] _desc) returns (bool r) {
        if (_device == address(0) || _desc.length == 0) {
            return false;
        }
        if (tx.origin == owner || tx.origin == container_executor(_device).get_owner()) {
            var existing_reg = devices[_device];
            bool exists = false;
            if (existing_reg.last_index != 0) {
                delete devices[_device];
                exists = true;
            }
            devices[_device] = Description({last_index:0});
            from_input(devices[_device],_desc);
            index[last_index] = _device;
            if (exists == false) {
                piggy_bank.mint(5);
                last_index += 1;
                add_to_mine(_device);
            }
            return true;
        } else {
            return false;
        }
    }
    function find_by_attributes(uint _start, uint _end, bytes32[] _filter) constant returns (address[] r) {
        if (_filter.length == 0 || _end < _start) {
            search_out.length = 0;
            return search_out;
        }
        uint max_size = _end - _start + 1;
        uint matches = 0;
        if (max_size == 0) {
            max_size = 20;
        }
        if (max_size > get_number_registered()) {
            max_size = get_number_registered();
        }
        search_out.length = max_size;
        uint all_devices = 0;
        bool done = false;
        while (all_devices < last_index && done == false) {
            var attribs = devices[index[all_devices]];
            uint a = 0;
            bool index_matches_filter = true;
            while (a < _filter.length && index_matches_filter == true) {
                if (_filter[a+1] != attribs.values[_filter[a]]) {
                    index_matches_filter = false;
                }
                a += 2;
            }
            if (index_matches_filter == true) {
                matches += 1;
                if (matches-1 >= _start && matches-1 <= _end) {
                    search_out[matches-1-_start] = index[all_devices];
                    if (matches-_start == max_size) {
                        done = true;
                    }
                }
            }
            all_devices += 1;
        }
        search_out.length = matches-_start;        
        return search_out;
    }
    function all_devices(uint _start, uint _end) constant returns (address[] r) {
        if (_end < _start) {
            search_out.length = 0;
            return search_out;
        }
        uint max_size = _end - _start + 1;
        if (max_size == 0 || max_size > get_number_registered()) {
            max_size = get_number_registered();
        }
        search_out.length = max_size;
        uint all_devices = _start;
        bool done = false;
        while (all_devices < last_index && done == false) {
            var attribs = devices[index[all_devices]];
            search_out[all_devices-_start] = index[all_devices];
            all_devices += 1;
            if (all_devices > _end) {
                done = true;
            }
        }
        if (done == false) {
            search_out.length = all_devices-_start;
        }
        return search_out;
    }
    function find_by_owner() constant returns (address[] r) {
        uint my_devices = 0;
        var mine = owned[tx.origin];
        search_out.length = mine.last_index;
        while (my_devices < mine.last_index) {
            search_out[my_devices] = mine.device[my_devices];
            my_devices += 1;
        }
        return search_out;
    }
    function add_to_mine(address _device) internal {
        var mine = owned[tx.origin];
        if (mine.last_index == 0) {
            owned[tx.origin] = MyDevices({last_index:0});
            mine = owned[tx.origin];
        }
        mine.device[mine.last_index] = _device;
        mine.last_index += 1;
    }
    function remove_from_mine(address _device) internal {
        var mine = owned[tx.origin];
        bool removed = false;
        if (mine.last_index != 0) {
            uint my_devices = 0;
            while (my_devices < mine.last_index) {
                if (mine.device[my_devices] == _device) {
                    delete mine.device[my_devices];
                    removed = true;
                } else if (removed == true) {
                    mine.device[my_devices-1] = mine.device[my_devices];
                }
                my_devices += 1;
            }
            delete mine.device[mine.last_index-1];
            mine.last_index -= 1;
        }
    }
    function get_description(address _device) constant returns (bytes32[] r) {
        return to_output(devices[_device]);
    }
    function deregister(address _device) returns (bool r) {
        if (_device == address(0)) {
            return false;
        }
        if (tx.origin == owner || tx.origin == container_executor(_device).get_owner()) { 
            delete devices[_device];
            uint i = 0;
            bool removed = false;
            while (i < last_index) {
                if (index[i] == _device) {
                    delete index[i];
                    removed = true;
                } else if (removed == true) {
                    index[i-1] = index[i];
                }
                i += 1;
            }
            delete index[last_index-1];
            last_index -= 1;
            remove_from_mine(_device);
            return true;
        } else {
            return false;
        }
    }
    function from_input(Description storage _new_desc, bytes32[] _desc) constant internal {
        _new_desc.last_index = 0;
        uint i = 0;
        while (i < _desc.length) {
            _new_desc.values[_desc[i]] = _desc[i+1];
            _new_desc.index[_new_desc.last_index] = _desc[i];
            _new_desc.last_index += 1;
            i += 2;
        }
        return;
    }
    function to_output(Description storage _d) constant internal returns (bytes32[] r) {
        desc_out.length = _d.last_index*2;
        uint i = 0;
        while (i < _d.last_index) {
            desc_out[i*2] = _d.index[i];
            desc_out[(i*2)+1] = _d.values[desc_out[i*2]];
            i += 1;
        }
        return desc_out;
    }
    function set_bank(address _bank) {
        if (owner == msg.sender) {
            piggy_bank = token_bank(_bank);
        }
    }
    function get_bank() constant returns (address r) {
        return piggy_bank;
    }
    function device_registry() {
        owner = msg.sender;
    }
    function get_owner() constant returns (address r) {
        return owner;
    }
    // Used to get rid of the contract
    function kill() {
        if (msg.sender == owner) suicide(owner);
    }
}