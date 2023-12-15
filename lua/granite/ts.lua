local M = {}

---
---@param node TSNode
---@param type string
---@return TSNode[]
M.get_all_children_of_type = function(node, type)
	local allnodes = {}
	for i = 1, node:child_count() do
		if node:child(i) and node:child(i):type() == type then
			table.insert(allnodes, node:child(i))
		end
	end
	return allnodes
end

---comment
---@param node TSNode
---@return string[]
M.get_text_of_node = function(node)
	local linestart, colstart, lineend, colend = node:range(false)
	return vim.api.nvim_buf_get_text(0, linestart, colstart, lineend, colend, {})
end


return M
