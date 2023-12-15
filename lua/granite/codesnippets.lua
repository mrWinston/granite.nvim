local ts = require("granite.ts")

function string:contains(sub)
	return self:find(sub, 1, true) ~= nil
end

function string:startswith(start)
	return self:sub(1, #start) == start
end

function string:endswith(ending)
	return ending == "" or self:sub(-#ending) == ending
end

---@class CodeBlock
---@field language string? language string of the block
---@field language_raw string?
---@field start_row integer
---@field end_row integer
---@field start_col integer
---@field end_col integer
---@field tsnode TSNode
---@field opts { string: string }
---@field text string[]?

local M = {}

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

---
---@param node TSNode
---@return CodeBlock?
local parse_codeblock = function(node)
	if not node or node:type() ~= "fenced_code_block" then
		return nil
	end

	local linestart, colstart, lineend, colend = node:range(false)
	---@type CodeBlock
	local codeblock = {
		tsnode = node,
		start_row = linestart,
		start_col = colstart,
		end_row = lineend,
		end_col = colend,
		opts = {},
	}

	if #ts.get_all_children_of_type(node, "info_string") == 1 then
		local info = ts.get_all_children_of_type(node, "info_string")[1]
		codeblock.language_raw = ts.get_text_of_node(info)[1]
	end

	local content_nodes = ts.get_all_children_of_type(node, "code_fence_content")
	if content_nodes then
		if content_nodes[1] then
			codeblock.text = ts.get_text_of_node(content_nodes[1])
      --- remove the last element, as it's always empty
      table.remove(codeblock.text)
		end
	end

	if codeblock.language_raw ~= nil then
		local split_info = mysplit(codeblock.language_raw, " ")
		if #split_info >= 1 then
			codeblock.language = split_info[1]
		end
		-- next, parse kv pairs for other configs
		for i = 2, #split_info, 1 do
			local key_value_split = mysplit(split_info[i], "=")
			if #key_value_split == 2 then
				codeblock.opts[key_value_split[1]] = key_value_split[2]
			end
		end
	end
	return codeblock
end

---comment
---@return CodeBlock[]
M.findAllCodeblocks = function()
	---@type CodeBlock[]
	local codeblocks = {}
	local parser = vim.treesitter.get_parser()
	local tree = parser:parse(true)[1]
	local cbquery = vim.treesitter.query.parse("markdown", "(fenced_code_block) @cb")
	for id, node, metadata in cbquery:iter_captures(tree:root(), 0, 0, -1) do
		local current = parse_codeblock(node)
		if current then
			table.insert(codeblocks, current)
		end
	end
	return codeblocks
end

local getCodeblockUnderCursor = function()
	local lineNum = vim.api.nvim_win_get_cursor(0)[1]
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
				codeblock.tsnode = node:parent()
			end
		end
		if lineNum >= codeblock.start_row and lineNum <= codeblock.end_row + 1 then
			return parse_codeblock(codeblock.tsnode)
		end
	end
end

---comment
---@param bufno integer
---@param codeblock CodeBlock
local function codeblockGenerateID(bufno, codeblock)
  codeblock.opts["ID"] = tostring(os.time(os.date("!*t")))
  M.writeCodeblock(bufno, codeblock)
end

---return codeblock env
---@param codeblock CodeBlock
---@return table<string, string>
local getEnvForCodeblock = function(codeblock)
	local envmap = {}

	local currentNode = codeblock.tsnode:parent()
	while currentNode and currentNode:type() == "section" do
		local child_code_blocks = ts.get_all_children_of_type(currentNode, "fenced_code_block")

		for i, codeblock_node in pairs(child_code_blocks) do
			local code_block = parse_codeblock(codeblock_node)
			if code_block and code_block.language == "env" then
				-- we found something :)
				for _, line in pairs(code_block.text) do
					local split = mysplit(line, "=")
					if #split == 2 then
						if envmap[split[1]] == nil then
							envmap[split[1]] = split[2]
						end
					end
				end
			end
		end

		currentNode = currentNode:parent()
	end

	return envmap
end


---
---@param bufno integer
---@param codeblock CodeBlock
M.writeCodeblock = function(bufno, codeblock)
  local out_lines = {}
  local header = "```" .. codeblock.language
  for k, v in pairs(codeblock.opts) do
    header = string.format("%s %s=%s", header, k, v)
  end

  table.insert(out_lines, header)
  for _, l in ipairs(codeblock.text) do
    table.insert(out_lines, l)
  end

  table.insert(out_lines, "```")
	vim.api.nvim_buf_set_lines(bufno, codeblock.start_row, codeblock.end_row, false, out_lines)

end

---
---@param bufno integer
---@param sourceCodeBlock CodeBlock
---@param targetCodeblock CodeBlock
---@param scriptfile string
---@return function(arg: vim.SystemCompleted)
M.getCompletionFunction = function(bufno, sourceCodeBlock, targetCodeblock, scriptfile)
	---@param arg vim.SystemCompleted
	return function(arg)
    local outlines = {}
    if arg.stderr ~= "" then
      outlines = mysplit(arg.stderr, "\n")
      table.insert(outlines, "---")
      for _, l in ipairs(mysplit(arg.stdout, "\n")) do
        table.insert(outlines, l)
      end
    else
		  outlines = mysplit(arg.stdout, "\n")
    end
    targetCodeblock.text = outlines
    targetCodeblock.opts["LAST_RUN"] = os.date("!%Y-%m-%dT%TZ")
		vim.schedule(function()
      M.writeCodeblock(bufno, targetCodeblock)
		end)
		os.remove(scriptfile)
	end
end

M.RunCodeblock = function()
	local curBuf = vim.api.nvim_get_current_buf()
	local codeblock = getCodeblockUnderCursor()
	if not codeblock then
		vim.print("Not in codeblock")
		return
	end

	if codeblock.opts["ID"] == nil then
		codeblockGenerateID(curBuf, codeblock)
	end

	local codeBlockEnv = getEnvForCodeblock(codeblock)

	-- find codeblock tagged with ID
	local allCodeblocks = M.findAllCodeblocks()
  local targetCodeblock = nil
	for _, c in pairs(allCodeblocks) do
		if c.opts["SOURCE"] == codeblock.opts["ID"] then
			targetCodeblock = c
		end
	end

	if targetCodeblock == nil then
    vim.print("Creating target block")
		vim.api.nvim_buf_set_lines(curBuf, codeblock.end_row + 1, codeblock.end_row + 1, false, {
			"```out SOURCE=" .. codeblock.opts["ID"],
      "",
			"```",
		})

    for _, c in pairs(M.findAllCodeblocks()) do
      if c.opts["SOURCE"] == codeblock.opts["ID"] then
        targetCodeblock = c
      end
    end
	end

  if targetCodeblock == nil then
    vim.print("Couldn't find target code block!")
    return
  end

	if codeblock.language == "sh" or codeblock.language == "bash" or codeblock.language == "zsh" then
		local scriptfilename = os.tmpname()
		local scriptfile = assert(io.open(scriptfilename, "w+"))
		scriptfile:write(table.concat(codeblock.text, "\n"))
		scriptfile:close()

		local completedFunc = M.getCompletionFunction(curBuf, codeblock, targetCodeblock, scriptfilename)

		local runArg = ""

		local runOpts = {
			text = true,
			env = codeBlockEnv,
		}

		if codeblock.opts["CWD"] and codeblock.opts["CWD"]:startswith("docker") then
			local cwdsplit = mysplit(codeblock.opts["CWD"], ":")

			if #cwdsplit ~= 2 then
				vim.print("No container name specified")
				return
			end
			local container_name = cwdsplit[2]

			local envarg = ""
			for key, value in pairs(codeBlockEnv) do
				envarg = string.format("%s  -e '%s=%s'", envarg, key, value)
			end

			if codeBlockEnv[container_name] ~= nil then
				container_name = codeBlockEnv[container_name]
			end
			runArg = string.format(
				"podman cp %s %s:/tmpscript && podman exec %s -i %s bash -c 'source /tmpscript'",
				scriptfilename,
				container_name,
				envarg,
				container_name
			)
		else
			runArg = ". " .. scriptfilename
			runOpts.cwd = codeblock.opts["CWD"]
		end

		vim.print("Running: " .. runArg)
		local out = vim.system({ "zsh", "-i", "-c", runArg }, runOpts, completedFunc)
	end
end

return M
