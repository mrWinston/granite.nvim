local mod = require("granite.todo")
local bufutils = require("granite.buffer")
local granite_telescope = require("granite.telescope")
local granite_tpl = require("granite.templating")
local granite_ts = require("granite.ts")
local a = require("plenary.async")
---@class NoteTemplate
---@field name string Name of the template
---@field parameters string[]? parameters for the
---@field path string Path to the template file
---@field output_folder string Where to create the template
---@field filename_template string? If set, use this template as the filename instead of asking

---@class Config
---@field granite_yaml string? Path to the granite_yaml file in knowledge base root

---@type Config?
local config = {}

local M = {}

---@type Config?
M.config = config

M.start_granite = function(host)
	return vim.fn.jobstart({ "granite.nvim" }, { rpc = true })
end

---@param opts Config?
-- you can define your setup function here. Usually configurations can be merged, accepting outside params and
-- you can also put some validation here for those.
M.setup = function(opts)
	M.config = vim.tbl_deep_extend("force", M.config, opts or {})
	-- make sure query output is highlighted with markdown
	--vim.treesitter.language.register("bash", "zsh")

	require("telescope").load_extension("granite_telescope")
	require("granite.cmp")
	if vim.g.loaded_granite_nvim then
		return
	end
	vim.g.loaded_granite_nvim = true
	vim.cmd([[
    function! s:RequireGranite(host) abort
      return jobstart(['granite.nvim'], {'rpc': v:true})
    endfunction
  
    call remote#host#Register('granite', 'x', function('s:RequireGranite'))

    call remote#host#RegisterPlugin('granite', '0', [
    \ {'type': 'function', 'name': 'GraniteGetAllTags', 'sync': 1, 'opts': {}},
    \ {'type': 'function', 'name': 'GraniteGetTemplates', 'sync': 1, 'opts': {}},
    \ {'type': 'function', 'name': 'GraniteGetTodos', 'sync': 1, 'opts': {}},
    \ {'type': 'function', 'name': 'GraniteInit', 'sync': 1, 'opts': {}},
    \ {'type': 'function', 'name': 'GraniteRenderTemplate', 'sync': 1, 'opts': {}},
    \ {'type': 'function', 'name': 'GraniteRunCodeblock', 'sync': 0, 'opts': {}},
    \ ])

  ]])

	local goConfig = {
		granite_yaml = M.config.granite_yaml,
		log_level = "debug",
	}
	vim.fn.GraniteInit(vim.fn.json_encode(goConfig))
end

M.new_note_from_template = a.void(function()
	local templates = vim.fn.json_decode(vim.fn.GraniteGetTemplates())
	---@type NoteTemplate
	local selectedTemplate = a.wrap(granite_telescope.choose_template, 2)(templates)

	local opts = {}
	local parameters = {}
	if selectedTemplate.filename_template and selectedTemplate.filename_template ~= "" then
		parameters = { table.unpack(selectedTemplate.parameters) }
	else
		parameters = { "filename", table.unpack(selectedTemplate.parameters) }
	end

	local selection = a.wrap(vim.ui.select, 3)({ "first", "second", "third" }, {})
	vim.print(selection)

	for _, param in ipairs(parameters) do
		opts[param] = a.wrap(vim.ui.input, 2)({ prompt = "Enter Value for " .. param })
	end

	opts["parent_file_path"] = vim.api.nvim_buf_get_name(0)

	local outpath = vim.fn.GraniteRenderTemplate(vim.fn.json_encode(selectedTemplate), vim.fn.json_encode(opts))

	vim.cmd("tabnew " .. outpath)
end)


M.RunCodeblock = require("granite.codesnippets").RunCodeblock

local function mysplit(inputstr, sep)
	if sep == nil then
		sep = "%s"
	end
	local t = {}
	for str in string.gmatch(inputstr, "([^" .. sep .. "]+)") do
		table.insert(t, str)
	end
	return t
end


M.ParseCodequeries = function()
	local parser = vim.treesitter.get_parser()
	local tree = parser:parse(true)[1]
	local cbquery =
		vim.treesitter.query.parse("markdown", "(fenced_code_block (info_string) @info (code_fence_content) @text)")
	for pattern, match, metadata in cbquery:iter_matches(tree:root(), 0, 0, -1, {}) do
		local codeblock = {}
		for id, node in pairs(match) do
			local name = cbquery.captures[id]
			local linestart, colstart, lineend, colend = node:range(false)
			local content = vim.api.nvim_buf_get_text(0, linestart, colstart, lineend, colend, {})

			if name == "info" then
				codeblock.language = content[1]
			end
			if name == "text" then
				codeblock.start_row = linestart
				codeblock.start_col = colstart
				codeblock.end_row = lineend
				codeblock.end_col = colend
			end
		end

		if codeblock.language:match("granite") then
			local filter = {
				due = codeblock.language:match('due="(%S-)"'),
				tag_query = codeblock.language:match('tags="(.-)"'),
			}
			local states_raw = codeblock.language:match('states="(%S-)"')
			if states_raw then
				filter.states = mysplit(states_raw, ",")
			end

			local todos = M.get_all_todos(filter)
			local inserted_lines = {}
			for _, todo in ipairs(todos) do
				local withouttask = string.gsub(todo.text, " #task", "")
				local tasklink = string.format("[link](%s)", bufutils.get_buffer_relative_path(0, todo.filename))
				table.insert(inserted_lines, string.format("- %s - %s", withouttask, tasklink))
			end
			vim.api.nvim_buf_set_lines(0, codeblock.start_row, codeblock.end_row, true, inserted_lines)
		end
	end
end

M.newHandwritten = function()
	local rootRaw = vim.fs.dirname(M.config.granite_yaml)
	if not rootRaw then
		return
	end
	local rootDir = vim.fs.normalize(rootRaw)
	local rnoteDir = vim.fs.joinpath(rootDir, "rnote")
	local basename = vim.fn.expand("%:t:r")
	local tplFile = vim.fs.joinpath(rnoteDir, "tpl.rnote")
	local targetFile = vim.fs.joinpath(rnoteDir, basename .. ".rnote")

	if vim.fn.filereadable(targetFile) == 0 then
		local success = vim.uv.fs_copyfile(tplFile, targetFile, {})
		if not success then
			vim.print(success)
			return
		end
	end

	local relative_path = bufutils.get_buffer_relative_path(0, targetFile)
	bufutils.write_at_cursor(0, string.format("[notes](%s)", relative_path))
end

---
---@param opts any
---@return Todo[]
M.get_all_todos = function(opts)
	local json_opts = vim.fn.json_encode(opts)
	local all_todos = vim.fn.json_decode(vim.fn.GraniteGetTodos(json_opts))
	return all_todos
end

-- TODO: Let this search for files returned by golang only
M.open_note = function()
	require("telescope.builtin").find_files({
		cwd = vim.fs.dirname(M.config.granite_yaml),
		find_command = { "fd", "md$" },
	})
end

M.extmarkTodos = function()
	local curbuf = vim.api.nvim_get_current_buf()
	local curwin = vim.api.nvim_get_current_win()
	local lines = vim.api.nvim_buf_get_lines(curbuf, 0, -1, true)
	local startLine = -1
	local endLine = -1
	local startstring = "- @GRANITETODO"

	for i, line in ipairs(lines) do
		if line:match("- @GRANITETODO") and startLine == -1 then
			startLine = i
		end
	end
	if startLine == -1 then
		vim.print("No matching found")
		return
	end
	local ns = vim.api.nvim_create_namespace("granite_extmarks")
	local virtlines = {}

	local todos = M.get_all_todos({ states = { "OPEN", "IN_PROGRESS" } })

	for _, todo in ipairs(todos) do
		table.insert(virtlines, { { string.format("- %s", todo.text), "Question" } })
	end
	vim.print(virtlines)
	vim.api.nvim_buf_set_extmark(curbuf, ns, startLine - 1, 0, {
		virt_lines = virtlines,
		virt_lines_above = true,
	})
end

M.insertTodos = function()
	local curbuf = vim.api.nvim_get_current_buf()
	local lines = vim.api.nvim_buf_get_lines(curbuf, 0, -1, true)
	local startLine = -1
	local endLine = -1
	local startstring = "- @GRANITETODO"
	local endstring = "- @GRANITEEND"

	for i, line in ipairs(lines) do
		if line:match("- @GRANITETODO") and startLine == -1 then
			startLine = i
		end
		if line:match("- @GRANITEEND") and endLine == -1 then
			endLine = i
		end
	end
	if startLine == -1 or endLine == -1 or startLine > endLine then
		vim.print("No matching found")
		return
	end
	vim.print("Start/end: " .. startLine .. "," .. endLine)
	local replacement = { startstring }

	local todos = M.get_all_todos({ states = { "OPEN", "IN_PROGRESS" } })

	for _, todo in ipairs(todos) do
		local withouttask = string.gsub(todo.text, " #task", "")
		local tasklink = string.format("[link](%s)", bufutils.get_buffer_relative_path(0, todo.filename))
		table.insert(replacement, string.format("- %s - %s", withouttask, tasklink))
	end

	table.insert(replacement, endstring)

	vim.api.nvim_buf_set_lines(curbuf, startLine - 1, endLine, true, replacement)
end

M.link_to_file = function()
	local actions = require("telescope.actions")
	local action_state = require("telescope.actions.state")

	local current_buffer = vim.api.nvim_get_current_buf()
	local current_window = vim.api.nvim_get_current_win()

	require("telescope.builtin").find_files({
		cwd = vim.fs.dirname(M.config.granite_yaml),
		find_command = { "fd", "md$" },
		attach_mappings = function(prompt_bufnr)
			actions.select_default:replace(function()
				actions.close(prompt_bufnr)
				local selection = action_state.get_selected_entry()
				local target_path = string.format("%s/%s", selection.cwd, selection[1])
				vim.print(target_path)

				local relative_path = bufutils.get_buffer_relative_path(current_buffer, target_path)
				vim.print(relative_path)
				bufutils.write_at_cursor(current_window, string.format("[](%s)", relative_path))
			end)
			return true
		end,
	})
end

return M
