local mem = require("mem")

function set(key, value)
    mem.set(key, value)
end

function get(key)
    return mem.get(key)
end
