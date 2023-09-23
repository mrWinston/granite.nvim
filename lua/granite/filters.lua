local M = {}

M.all = function(todos)
	return todos
end

---filter todo's that are not done yet
---@param todos Todo[]
---@return Todo[]
M.not_done = function(todos)
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
M.done = function(todos)
	local newTodos = {}
	for _, todo in pairs(todos) do
		if todo.state == NOTE_STATE.DONE then
			table.insert(newTodos, todo)
		end
	end
	return newTodos
end

---filter todo's that are done
---@param todos Todo[]
---@return Todo[]
M.due_today = function(todos)
	local newTodos = {}
	local today = os.time({ year = os.date("*t").year, month = os.date("*t").month, day = os.date("*t").day })
	local matchpattern = "(%d+)%-(%d+)%-(%d+)"

	for _, todo in pairs(todos) do
		if todo.due_date then
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
