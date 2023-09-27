local M = {}

---Parse the given template config and return a list of NoteTemplates
---@param template_config_path string The full path to the template config file. either yaml or json
---@return NoteTemplate[]
M.get_templates_from_config = function(template_config_path)
	---@type NoteTemplate[]
	local templates = {}
	if vim.fn.filereadable(template_config_path) == 0 then
		error(string.format("Template Config '%s' can't be read.", template_config_path))
	end
	local json_out
	if string.find(template_config_path, "yaml$") then
		local out = vim.system({ "yq", ".", template_config_path, "--output-format", "json" }, { text = true }):wait()
		if out.code ~= 0 then
			error(string.format("Error reading yaml template config", template_config_path, out.stderr))
		end
		json_out = out.stdout
	else
		json_out = vim.fn.readfile(template_config_path)
	end
	templates = vim.fn.json_decode(json_out)
	vim.print(templates)
	return templates
end

M.render_template = function(template_path, opts)
	---@type string
	local json_data = vim.fn.json_encode(opts)

	local out = vim.system({ "tera", "--stdin", "--template", template_path }, { text = true, stdin = json_data })
		:wait()
	if out.code ~= 0 then
		error(string.format("Error rendering template %s: %s", template_path, out.stderr))
	end
	return out.stdout
end

-- workflow for template:
-- call template function
-- select template
-- select name
-- select optional vars
-- pass all vars and additional env stuff to the render function

return M
