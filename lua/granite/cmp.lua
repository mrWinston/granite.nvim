local source = {}

source.new = function()
	local self = setmetatable({}, { __index = source })
	return self
end

function source.is_available()
	return vim.bo.filetype == "markdown"
end

function source.get_debug_name()
	return "granite_cmp"
end

function source.get_trigger_characters()
	return { "#" }
end

---Invoke completion (required).
---@param params cmp.SourceCompletionApiParams
---@param callback fun(response: lsp.CompletionResponse|nil)
function source.complete(_, params, callback)
	local allTags = vim.fn.GraniteGetAllTags()
	local items = {}
	for _, tag in ipairs(allTags) do
		table.insert(items, { label = tag })
	end
	callback({ items = items, isIncomplete = false })
end

function source.resolve(_, completion_item, callback)
	callback(completion_item)
end

function source.execute(_, completion_item, callback)
	callback(completion_item)
end

vim.print("Init custom cmp")
---Register your source to nvim-cmp.
require("cmp").register_source("granite_cmp", source)
