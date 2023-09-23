local M = {}

---Write the given text at the current cursor position in the window given
---@param winnr integer number of the window to write to. 0 refers to the current window
---@param text string text to write at the cursor position
M.write_at_cursor = function(winnr, text)
	local bufnr = vim.api.nvim_win_get_buf(winnr)
	local row, col = unpack(vim.api.nvim_win_get_cursor(winnr))
	vim.api.nvim_buf_set_text(bufnr, row - 1, col, row - 1, col, { text })
end

M.get_buffer_relative_path = function(bufnr, target)
	local relative_to = vim.api.nvim_buf_get_name(bufnr)
	local out = vim.system(
		{ "zsh", "-c", string.format("realpath --relative-to=%s %s", relative_to, target) },
		{ text = true }
	)
		:wait()
	return out.stdout:gsub("\n", "")
end

return M
