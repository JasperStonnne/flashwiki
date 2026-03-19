export type Locale = "en" | "zh";

export interface Dictionary {
  slash: {
    heading1: { title: string; description: string };
    heading2: { title: string; description: string };
    heading3: { title: string; description: string };
    bulletList: { title: string; description: string };
    numberedList: { title: string; description: string };
    taskList: { title: string; description: string };
    blockquote: { title: string; description: string };
    codeBlock: { title: string; description: string };
    horizontalRule: { title: string; description: string };
    table: { title: string; description: string };
    image: { title: string; description: string };
    video: { title: string; description: string };
    mathBlock: { title: string; description: string };
    plantuml: { title: string; description: string };
  };
  slashMenu: {
    groups: {
      text: string;
      list: string;
      block: string;
      media: string;
      advanced: string;
    };
    noResults: string;
  };
  contextMenu: {
    addLink: string;
    searchFor: string;
    format: string;
    paragraph: string;
    insert: string;
    bold: string;
    italic: string;
    strikethrough: string;
    highlight: string;
    code: string;
    math: string;
    clearFormatting: string;
    bulletList: string;
    numberedList: string;
    taskList: string;
    heading1: string;
    heading2: string;
    heading3: string;
    heading4: string;
    heading5: string;
    heading6: string;
    body: string;
    quote: string;
    image: string;
    video: string;
    table: string;
    horizontalRule: string;
    codeBlock: string;
    mathBlock: string;
    plantumlBlock: string;
    cut: string;
    copy: string;
    paste: string;
    pasteAsPlainText: string;
    selectAll: string;
  };
  placeholder: {
    default: string;
    heading: string;
  };
  mathBlock: {
    enterLatex: string;
    clickToAdd: string;
    invalidLatex: string;
  };
  plantumlBlock: {
    clickToAdd: string;
    rendering: string;
    failedToRender: string;
    invalidSyntax: string;
  };
  tabHandler: {
    cannotIndent: string;
  };
  viewToggle: {
    richText: string;
    markdownSource: string;
  };
  tableMenu: {
    addRowBefore: string;
    addRowAfter: string;
    deleteRow: string;
    addColBefore: string;
    addColAfter: string;
    deleteCol: string;
    deleteTable: string;
    row: string;
    col: string;
    table: string;
  };
  dragHandle: {
    dragToMove: string;
  };
  editor: {
    defaultContent: string;
  };
  langToggle: {
    label: string;
  };
  exportButton: {
    title: string;
    exporting: string;
    videoLabel: string;
    scanToWatch: string;
  };
}

export interface LocaleRef {
  current: Dictionary;
}
