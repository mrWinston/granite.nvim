local M = {}

local telescope = require("telescope")
local builtin = require("telescope.builtin")
local pickers = require("telescope.pickers")
local finders = require("telescope.finders")
local conf = require("telescope.config").values
local actions = require("telescope.actions")
local action_state = require("telescope.actions.state")

---Open a new telescope picker to choose a template. Then call function "after" with the chosen template as parameter
---@param templates NoteTemplate[]
---@param after fun(selected: NoteTemplate)
M.choose_template = function(templates, after)
	pickers.new({}, {
		prompt_title = "templates",
		finder = finders.new_table({
			results = templates,
			entry_maker = function(entry)
				return {
					value = entry,
					display = entry.name,
					ordinal = entry.name,
				}
			end,
		}),
		sorter = conf.generic_sorter({}),
		attach_mappings = function(prompt_bufnr, map)
			actions.select_default:replace(function()
				local selection = action_state.get_selected_entry()
				actions.close(prompt_bufnr)
        after(selection.value)
			end)
			return true
		end,
	}):find()
end

return M
