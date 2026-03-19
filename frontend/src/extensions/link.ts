import Link from "@tiptap/extension-link";

const ConfiguredLink = Link.configure({
  openOnClick: false,
  autolink: true,
  linkOnPaste: true,
});

export { ConfiguredLink as Link };
