import { CodeBlockLowlight } from "@tiptap/extension-code-block-lowlight";
import { common, createLowlight } from "lowlight";

const lowlight = createLowlight(common);

const ConfiguredCodeBlockLowlight = CodeBlockLowlight.configure({
  lowlight,
});

export { ConfiguredCodeBlockLowlight as CodeBlockLowlight };
