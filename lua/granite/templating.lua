local M = {}

M.render_template = function(template_path, opts)
  ---@type string
  local json_data = vim.fn.json_encode(opts)

	local out = vim.system({ "mustache", template_path }, { text = true, stdin = json_data }):wait()

  return out.stdout
end

return M
