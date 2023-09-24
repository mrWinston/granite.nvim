local M = {}

M.render_template = function(template_path, opts)
  ---@type string
  local json_data = vim.fn.json_encode(opts)

	local out = vim.system({ "tera", "--stdin", "--template", template_path }, { text = true, stdin = json_data }):wait()
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
