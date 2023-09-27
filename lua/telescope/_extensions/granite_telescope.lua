local pickers = require("telescope.pickers")
local finders = require("telescope.finders")
local conf = require("telescope.config").values

local actions = require("telescope.actions")
local action_state = require("telescope.actions.state")
local granite = require("granite")

return require("telescope").register_extension({
	setup = function(ext_config, config)
		-- access extension config and user config
	end,
	exports = {
		granite_telescope = function(opts)
			opts = opts or {}
			pickers
				.new(opts, {
					prompt_title = "todos",
					finder = finders.new_table({
						results = granite.get_all_todos(),
						entry_maker = function(entry)
							return {
								value = entry,
								display = entry.text,
								ordinal = entry.text,
							}
						end,
					}),
					sorter = conf.generic_sorter(opts),
					attach_mappings = function(prompt_bufnr, map)
						actions.select_default:replace(function()
							local selection = action_state.get_selected_entry()
		          actions.close(prompt_bufnr)
              vim.cmd("e ".. selection.value.filename)
              vim.cmd(tostring(selection.value.lnum))
						end)
						actions.file_tab:replace(function()
							local selection = action_state.get_selected_entry()
		          actions.close(prompt_bufnr)
              vim.cmd("tabnew ".. selection.value.filename)
              vim.cmd(tostring(selection.value.lnum))
						end)
						actions.file_split:replace(function()
							local selection = action_state.get_selected_entry()
		          actions.close(prompt_bufnr)
              vim.cmd("split ".. selection.value.filename)
              vim.cmd(tostring(selection.value.lnum))
						end)
						actions.file_vsplit:replace(function()
							local selection = action_state.get_selected_entry()
		          actions.close(prompt_bufnr)
              vim.cmd("vsplit ".. selection.value.filename)
              vim.cmd(tostring(selection.value.lnum))
						end)
						return true
					end,
				})
				:find()
		end,
	},
})
