// Input rule regex for inline math: matches $...$
// Captures the content between the dollar signs
export const inputRegex = /(?:^|[^$])\$([^$]+)\$$/;
