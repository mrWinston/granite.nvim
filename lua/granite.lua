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
---@field granite_yaml string? Path to the granite_yaml file in knowledge base root

---@type Config?
local config = {}

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
    \ {'type': 'function', 'name': 'GraniteRenderTemplate', 'sync': 1, 'opts': {}},
    \ ])
  ]])

	local goConfig = {
		GraniteYaml = M.config.granite_yaml,
		LogLevel = "debug",
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

	for _, param in ipairs(parameters) do
		opts[param] = a.wrap(vim.ui.input, 2)({ prompt = "Enter Value for " .. param })
	end

	opts["parent_file_path"] = vim.api.nvim_buf_get_name(0)

	local outpath = vim.fn.GraniteRenderTemplate(vim.fn.json_encode(selectedTemplate), vim.fn.json_encode(opts))

	vim.cmd("tabnew " .. outpath)
end)

---Returns all todos found
---@return Todo[]
M.get_all_todos = function()
	local all_todos = vim.fn.json_decode(vim.fn.GraniteGetTodos())
	return all_todos
end

-- TODO: Let this search for files returned by golang only
M.open_note = function()
	require("telescope.builtin").find_files({
		cwd = vim.fs.dirname(M.config.granite_yaml),
		find_command = { "fd", "md$" },
	})
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
