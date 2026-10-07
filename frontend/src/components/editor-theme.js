// Forked from https://github.com/fsegurai/codemirror-themes
// MIT License - Copyright (c) 2025 fsegurai

import { EditorView } from '@codemirror/view';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { tags } from '@lezer/highlight';

// VSCode Light theme color definitions
const background = '#ffffff';
const foreground = '#383a42';
const caret = '#000000';
const selection = '#add6ff';
const selectionMatch = '#a8ac94';
const lineHighlight = '#99999926';
const gutterBackground = '#ffffff';
const gutterForeground = '#0055d4';
const gutterActiveForeground = '#0b216f';
const keywordColor = '#0055d4';
const controlKeywordColor = '#af00db';
const variableColor = '#e45649';
const classTypeColor = '#0055d4';
const functionColor = '#795e26';
const numberColor = '#098658';
const operatorColor = '#383a42';
const regexpColor = '#af00db';
const stringColor = '#50a14f';
const commentColor = '#999';
const linkColor = '#0055d4';
const invalidColor = '#e45649';

// Define the editor theme styles for VSCode Light
const vsCodeLightTheme = /* @__PURE__ */EditorView.theme({
  '&': {
    color: foreground,
    backgroundColor: background,
    fontFamily: 'Menlo, Monaco, Consolas, "Andale Mono", "Ubuntu Mono", "Courier New", monospace',
  },
  '.cm-content': {
    caretColor: caret,
  },
  '.cm-cursor, .cm-dropCursor': {
    borderLeftColor: caret,
  },
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: selection,
  },
  '.cm-searchMatch': {
    backgroundColor: selectionMatch,
    outline: `1px solid ${lineHighlight}`,
  },
  '.cm-activeLine': {
    backgroundColor: lineHighlight,
  },
  '.cm-gutters': {
    backgroundColor: gutterBackground,
    color: gutterForeground,
  },
  '.cm-activeLineGutter': {
    color: gutterActiveForeground,
  },
}, { dark: false });
// Define the highlighting style for code in the VSCode Light theme
const vsCodeLightHighlightStyle = /* @__PURE__ */HighlightStyle.define([
  {
    tag: [
      tags.keyword,
      tags.operatorKeyword,
      tags.modifier,
      tags.color,
      /* @__PURE__ */tags.constant(tags.name),
      /* @__PURE__ */tags.standard(tags.name),
      /* @__PURE__ */tags.standard(tags.tagName),
      /* @__PURE__ */tags.special(tags.brace),
      tags.atom,
      tags.bool,
      /* @__PURE__ */tags.special(tags.variableName),
    ],
    color: keywordColor,
  },
  { tag: [tags.moduleKeyword, tags.controlKeyword], color: controlKeywordColor },
  {
    tag: [
      tags.name,
      tags.deleted,
      tags.character,
      tags.macroName,
      tags.propertyName,
      tags.variableName,
      tags.labelName,
      /* @__PURE__ */tags.definition(tags.name),
    ],
    color: variableColor,
  },
  { tag: tags.heading, fontWeight: 'bold', color: variableColor },
  {
    tag: [
      tags.typeName,
      tags.className,
      tags.tagName,
      tags.number,
      tags.changed,
      tags.annotation,
      tags.self,
      tags.namespace,
    ],
    color: classTypeColor,
  },
  {
    tag: [/* @__PURE__ */tags.function(tags.variableName), /* @__PURE__ */tags.function(tags.propertyName)],
    color: functionColor,
  },
  { tag: [tags.number], color: numberColor },
  {
    tag: [tags.operator, tags.punctuation, tags.separator, tags.url, tags.escape, tags.regexp],
    color: operatorColor,
  },
  { tag: [tags.regexp], color: regexpColor },
  {
    tag: [/* @__PURE__ */tags.special(tags.string), tags.processingInstruction, tags.string, tags.inserted],
    color: stringColor,
  },
  { tag: [tags.meta, tags.comment], color: commentColor },
  { tag: tags.link, color: linkColor, textDecoration: 'underline' },
  { tag: tags.invalid, color: invalidColor },
  { tag: tags.strong, fontWeight: 'bold' },
  { tag: tags.emphasis, fontStyle: 'italic' },
  { tag: tags.strikethrough, textDecoration: 'line-through' },
]);
// Extension to enable the VSCode Light theme (both the editor theme and the highlight style)
const vsCodeLight = [
  vsCodeLightTheme,
  /* @__PURE__ */syntaxHighlighting(vsCodeLightHighlightStyle),
];

// VSCode Dark theme color definitions (matches the admin dark palette)
const darkBackground = '#0F172A';
const darkForeground = '#CBD5E1';
const darkCaret = '#F59E0B';
const darkSelection = 'rgba(245, 158, 11, 0.22)';
const darkSelectionMatch = 'rgba(127, 178, 229, 0.35)';
const darkLineHighlight = 'rgba(255, 255, 255, 0.04)';
const darkGutterBackground = '#0F172A';
const darkGutterForeground = '#64748B';
const darkGutterActiveForeground = '#F59E0B';
const darkKeywordColor = '#7FB2E5';
const darkControlKeywordColor = '#C4B5FD';
const darkVariableColor = '#F87171';
const darkClassTypeColor = '#93C5FD';
const darkFunctionColor = '#FBBF24';
const darkNumberColor = '#6EE7B7';
const darkOperatorColor = '#94A3B8';
const darkRegexpColor = '#C4B5FD';
const darkStringColor = '#6EE7B7';
const darkCommentColor = '#64748B';
const darkLinkColor = '#7FB2E5';
const darkInvalidColor = '#F87171';

// Define the editor theme styles for VSCode Dark
const vsCodeDarkTheme = /* @__PURE__ */EditorView.theme({
  '&': {
    color: darkForeground,
    backgroundColor: darkBackground,
    fontFamily: 'Menlo, Monaco, Consolas, "Andale Mono", "Ubuntu Mono", "Courier New", monospace',
  },
  '.cm-content': {
    caretColor: darkCaret,
  },
  '.cm-cursor, .cm-dropCursor': {
    borderLeftColor: darkCaret,
  },
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: darkSelection,
  },
  '.cm-searchMatch': {
    backgroundColor: darkSelectionMatch,
    outline: `1px solid ${darkLineHighlight}`,
  },
  '.cm-activeLine': {
    backgroundColor: darkLineHighlight,
  },
  '.cm-gutters': {
    backgroundColor: darkGutterBackground,
    color: darkGutterForeground,
    borderRight: '1px solid #1E293B',
  },
  '.cm-activeLineGutter': {
    color: darkGutterActiveForeground,
  },
}, { dark: true });

// Define the highlighting style for code in the VSCode Dark theme
const vsCodeDarkHighlightStyle = /* @__PURE__ */HighlightStyle.define([
  {
    tag: [
      tags.keyword,
      tags.operatorKeyword,
      tags.modifier,
      tags.color,
      /* @__PURE__ */tags.constant(tags.name),
      /* @__PURE__ */tags.standard(tags.name),
      /* @__PURE__ */tags.standard(tags.tagName),
      /* @__PURE__ */tags.special(tags.brace),
      tags.atom,
      tags.bool,
      /* @__PURE__ */tags.special(tags.variableName),
    ],
    color: darkKeywordColor,
  },
  { tag: [tags.moduleKeyword, tags.controlKeyword], color: darkControlKeywordColor },
  {
    tag: [
      tags.name,
      tags.deleted,
      tags.character,
      tags.macroName,
      tags.propertyName,
      tags.variableName,
      tags.labelName,
      /* @__PURE__ */tags.definition(tags.name),
    ],
    color: darkVariableColor,
  },
  { tag: tags.heading, fontWeight: 'bold', color: darkVariableColor },
  {
    tag: [
      tags.typeName,
      tags.className,
      tags.tagName,
      tags.number,
      tags.changed,
      tags.annotation,
      tags.self,
      tags.namespace,
    ],
    color: darkClassTypeColor,
  },
  {
    tag: [/* @__PURE__ */tags.function(tags.variableName), /* @__PURE__ */tags.function(tags.propertyName)],
    color: darkFunctionColor,
  },
  { tag: [tags.number], color: darkNumberColor },
  {
    tag: [tags.operator, tags.punctuation, tags.separator, tags.url, tags.escape, tags.regexp],
    color: darkOperatorColor,
  },
  { tag: [tags.regexp], color: darkRegexpColor },
  {
    tag: [/* @__PURE__ */tags.special(tags.string), tags.processingInstruction, tags.string, tags.inserted],
    color: darkStringColor,
  },
  { tag: [tags.meta, tags.comment], color: darkCommentColor },
  { tag: tags.link, color: darkLinkColor, textDecoration: 'underline' },
  { tag: tags.invalid, color: darkInvalidColor },
  { tag: tags.strong, fontWeight: 'bold' },
  { tag: tags.emphasis, fontStyle: 'italic' },
  { tag: tags.strikethrough, textDecoration: 'line-through' },
]);

// Extension to enable the VSCode Dark theme (both the editor theme and the highlight style)
const vsCodeDark = [
  vsCodeDarkTheme,
  /* @__PURE__ */syntaxHighlighting(vsCodeDarkHighlightStyle),
];

export {
  vsCodeLight, vsCodeLightHighlightStyle, vsCodeLightTheme,
  vsCodeDark, vsCodeDarkHighlightStyle, vsCodeDarkTheme,
};
