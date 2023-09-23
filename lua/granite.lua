local mod = require("granite.todo")
---@class NoteTemplate
---@field name string Name of the template
---@field template_path string Path to the template file
---@field create_folder string Where to create the template

---@class Config
---@field knowledge_base_path string? Path to the knowledge base folder
---@field templates NoteTemplate[]? Templates
---@field default_note_folder string? Templates
---@field todo_tag string? Tag for todo items

---@type Config
local config = {
	knowledge_base_path = "~/knowledge",
	default_note_folder = "notes",
	templates = {},
	todo_tag = "#task",
}

---@class Knowledge
local M = {}

---@type Config
M.config = config

---@param args Config?
-- you can define your setup function here. Usually configurations can be merged, accepting outside params and
-- you can also put some validation here for those.
M.setup = function(args)
	M.config = vim.tbl_deep_extend("force", M.config, args or {})
end

M.Note = function()
	vim.ui.input({ prompt = "Note name:" }, function(input)
		if not input then
			return
		end
		if not string.match(input, ".md$") then
			input = input .. ".md"
		end
		local fullpath = string.format(
			"%s/%s/%s",
			vim.fs.normalize(M.config.knowledge_base_path),
			vim.fs.normalize(M.config.default_note_folder),
			input
		)
		vim.cmd("tabnew " .. fullpath)
	end)
end

M.scan = function()
	local todos = mod.scan_todos(M.config.knowledge_base_path, M.config.todo_tag)

	vim.fn.setqflist(todos, "r")
	vim.cmd("copen")
end

---
---@param filter fun(filter: Todo[]): Todo[]
M.show_tasks = function(filter)
	local todos = mod.scan_todos(M.config.knowledge_base_path, M.config.todo_tag)
	if filter then
		todos = filter(todos)
	end

	vim.fn.setqflist(todos, "r")
	vim.cmd("copen")
end

---Returns all todos found
---@return Todo[]
M.get_all_todos = function()
	return mod.scan_todos(M.config.knowledge_base_path, M.config.todo_tag)
end

---filter todo's that are not done yet
---@param todos Todo[]
---@return Todo[]
M.filter_not_done = function(todos)
	local newTodos = {}
	for _, todo in pairs(todos) do
		if todo.state ~= NOTE_STATE.DONE then
			table.insert(newTodos, todo)
		end
	end
	return newTodos
end

---filter todo's that are done
---@param todos Todo[]
---@return Todo[]
M.filter_done = function(todos)
	local newTodos = {}
	for _, todo in pairs(todos) do
		if todo.state == NOTE_STATE.DONE then
			table.insert(newTodos, todo)
		end
	end
	return newTodos
end

M.open_note = function()
	require("telescope.builtin").find_files({
		cwd = M.config.knowledge_base_path,
    find_command = {"fd", "md$"},
	})
end

M.link_to_file = function()
	local actions = require("telescope.actions")
	local action_state = require("telescope.actions.state")
	local current_buffer_name = vim.fn.expand("%:p:h")
	local row, col = unpack(vim.api.nvim_win_get_cursor(0))
	local curBuf = vim.api.nvim_get_current_buf()
	require("telescope.builtin").find_files({
		cwd = M.config.knowledge_base_path,
    find_command = {"fd", "md$"},
		attach_mappings = function(prompt_bufnr, map)
			actions.select_default:replace(function()
				actions.close(prompt_bufnr)
				local selection = action_state.get_selected_entry()
				local fullpath = string.format("%s/%s", selection.cwd, selection[1])
				local relto = current_buffer_name

				local out = vim.system(
					{ "zsh", "-c", string.format("realpath --relative-to=%s %s", relto, fullpath) },
					{ text = true }
				):wait()

				vim.api.nvim_buf_set_text(
					curBuf,
					row - 1,
					col,
					row - 1,
					col,
					{ string.format("[](%s)", out.stdout:gsub("[\n\r]", "")) }
				)
			end)
			return true
		end,
	})
end

---filter todo's that are done
---@param todos Todo[]
---@return Todo[]
M.filter_due_today = function(todos)
	local newTodos = {}
	local today = os.time({ year = os.date("*t").year, month = os.date("*t").month, day = os.date("*t").day })
	for _, todo in pairs(todos) do
		if todo.due_date then
			local matchpattern = "(%d+)%-(%d+)%-(%d+)"
			local due_year, due_month, due_day = todo.due_date:match(matchpattern)
			local due_time = os.time({ year = due_year, month = due_month, day = due_day })
			if due_time == today then
				table.insert(newTodos, todo)
			end
		end
	end
	return newTodos
end

return M
