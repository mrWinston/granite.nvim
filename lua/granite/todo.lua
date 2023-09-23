local M = {}

local set_table_default = function(table, default)
	local mt = {
		__index = function()
			return default
		end,
	}
	setmetatable(table, mt)
end

---@enum note_state
NOTE_STATE = {
	OPEN = 0,
	IN_PROGRESS = 1,
	DONE = 2,
}

---@type {[note_state]: string}
STATE_TO_TYPE_CHAR = {
	[NOTE_STATE.OPEN] = "O",
	[NOTE_STATE.IN_PROGRESS] = "P",
	[NOTE_STATE.DONE] = "D",
}

STATE_MAP = {
	["[]"] = NOTE_STATE.OPEN,
	["[ ]"] = NOTE_STATE.OPEN,
	["[-]"] = NOTE_STATE.IN_PROGRESS,
	["[/]"] = NOTE_STATE.IN_PROGRESS,
	["[x]"] = NOTE_STATE.DONE,
	["[X]"] = NOTE_STATE.DONE,
}
set_table_default(STATE_MAP, NOTE_STATE.OPEN)

---@class Todo
---@field tags string[] Tags of the todo
---@field due_date string Due date
---@field text string Text of the todo
---@field state note_state
---@field filename string
---@field lnum number
---@field type string

---Returns weather or not the given line is a todo item line
---@param markdown_line string
---@param todo_tag string
---@return boolean
M.is_line_todo = function(markdown_line, todo_tag)
	if string.find(markdown_line, "^.*%[[ -xX]?%]") then
		if string.find(markdown_line, todo_tag) then
			return true
		end
	end
	return false
end

---
---@param line string
---@param todo_tag string
---@return Todo|nil
M.line_to_todo = function(line, todo_tag)
	if not M.is_line_todo(line, todo_tag) then
		return nil
	end
	local tags = {}
	for t in string.gmatch(line, "(#%a+)") do
		table.insert(tags, t)
	end
	local due_date = string.match(line, "(%d%d%d%d%-%d%d%-%d%d)")
	local text = string.match(line, "^.*(%[[ -xX]?%].*)$")
	local stateStr = string.match(line, "^.*(%[[ -xX]?%]).*$")
	local state = STATE_MAP[stateStr]
	return {
		due_date = due_date,
		tags = tags,
		text = text,
		state = STATE_MAP[stateStr],
		type = STATE_TO_TYPE_CHAR[state],
	}
end

M.extract_todos = function(markdown_file_lines, todo_tag)
	---@type Todo[]
	local todos = {}
	for i, line in pairs(markdown_file_lines) do
		local todo = M.line_to_todo(line, todo_tag)
		if todo then
			todo.lnum = i
			table.insert(todos, todo)
		end
	end
	return todos
end

---Scan all todo's from path
---@param knowledge_base_path string path to the knowledge base to parse
---@param todo_tag string path to the knowledge base to parse
---@return Todo[]
M.scan_todos = function(knowledge_base_path, todo_tag)
	local path = vim.fs.normalize(knowledge_base_path) .. "/**/*.md"
	local markdown_file_paths = vim.fn.glob(path, false, true)
	local all_todos = {}
	for _, md_path in pairs(markdown_file_paths) do
		local todos = M.extract_todos(vim.fn.readfile(md_path), todo_tag)
		for _, todo in pairs(todos) do
			todo.filename = md_path
			table.insert(all_todos, todo)
		end
	end
	return all_todos
end

return M
