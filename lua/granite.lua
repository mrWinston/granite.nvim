local mod = require("granite.todo")
local bufutils = require("granite.buffer")
local granite_telescope = require("granite.telescope")
local granite_tpl = require("granite.templating")
local a = require("plenary.async")
---@class NoteTemplate
---@field name string Name of the template
---@field parameters string[]? parameters for the
---@field path string Path to the template file
---@field output_folder string Where to create the template
---@field filename_template string? If set, use this template as the filename instead of asking

---@class Config
---@field knowledge_base_path string? Path to the knowledge base folder
---@field template_config string Templates
---@field todo_tag string? Tag for todo items

---@type Config?
local config = {
	knowledge_base_path = "~/knowledge",
	default_note_folder = "notes",
	templates = {},
	template_config = "",
	todo_tag = "#task",
}

---@class Knowledge
local M = {}

---@type Config?
M.config = config

M.start_granite = function(host)
	return vim.fn.jobstart({ "granite.nvim" }, { rpc = true })
end

---@param args Config?
-- you can define your setup function here. Usually configurations can be merged, accepting outside params and
-- you can also put some validation here for those.
M.setup = function(args)
	M.config = vim.tbl_deep_extend("force", M.config, args or {})
	if vim.g.loaded_granite_nvim then
		return
	end
	vim.g.loaded_granite_nvim = true
	--  vim.fn["remote#host#Register"]("granite", "x", M.start_granite)
	vim.cmd([[
    function! s:RequireGranite(host) abort
      return jobstart(['granite.nvim'], {'rpc': v:true})
    endfunction
  
    call remote#host#Register('granite', 'x', function('s:RequireGranite'))
    call remote#host#RegisterPlugin('granite', '0', [
    \ {'type': 'function', 'name': 'GraniteGetTemplates', 'sync': 1, 'opts': {}},
    \ {'type': 'function', 'name': 'GraniteGetTodos', 'sync': 1, 'opts': {}},
    \ {'type': 'function', 'name': 'GraniteInit', 'sync': 1, 'opts': {}},
    \ ])
  ]])

	local goConfig = {
		RootPath = vim.fs.normalize(M.config.knowledge_base_path),
		TodoTag = M.config.todo_tag,
		TemplateConfigPath = M.config.template_config,
		LogLevel = "debug",
	}
	vim.fn.GraniteInit(vim.fn.json_encode(goConfig))
end

M.new_note_from_template = a.void(function()
	local tx, rx = a.control.channel.oneshot()
	local full_template_config_path =
		vim.fs.normalize(vim.fs.joinpath(M.config.knowledge_base_path, M.config.template_config))
	local templates = granite_tpl.get_templates_from_config(full_template_config_path)
	granite_telescope.choose_template(templates, function(selected)
		tx(selected)
	end)
	---@type NoteTemplate
	local selected = rx()

	-- validate template path
	local full_template_path = vim.fs.normalize(vim.fs.joinpath(M.config.knowledge_base_path, selected.path))
	if vim.fn.filereadable(full_template_path) == 0 then
		error(string.format("Template file '%s' can't be read.", full_template_path))
	end

	local opts = {}
	local parameters = {}
	parameters = { "filename", table.unpack(selected.parameters) }

	for _, param in ipairs(parameters) do
		tx, rx = a.control.channel.oneshot()
		vim.ui.input({ prompt = "Enter Value for " .. param }, function(input)
			tx(input)
		end)
		local value = rx()
		opts[param] = value
	end
	opts["parent_file_path"] = vim.api.nvim_buf_get_name(0)


	local tpl_string = granite_tpl.render_template(full_template_path, opts)
	local outpath =
		vim.fs.normalize(vim.fs.joinpath(M.config.knowledge_base_path, selected.output_folder, opts.filename))
	vim.fn.writefile(vim.split(tpl_string, "\n"), outpath, "")
	vim.cmd("tabnew " .. outpath)
end)

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
	local all_todos = vim.fn.json_decode(vim.fn.GraniteGetTodos())
	return all_todos
end

M.open_note = function()
	require("telescope.builtin").find_files({
		cwd = M.config.knowledge_base_path,
		find_command = { "fd", "md$" },
	})
end

M.link_to_file = function()
	local actions = require("telescope.actions")
	local action_state = require("telescope.actions.state")

	local current_buffer = vim.api.nvim_get_current_buf()
	local current_window = vim.api.nvim_get_current_win()

	require("telescope.builtin").find_files({
		cwd = M.config.knowledge_base_path,
		find_command = { "fd", "md$" },
		attach_mappings = function(prompt_bufnr, map)
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
