/**
 * CodeMirror 6 autocompletion source for [[wikilinks]].
 * Triggers after typing "[[" and filters the note list as the user types.
 */
import {
  autocompletion,
  type CompletionContext,
  type CompletionResult,
  type Completion,
} from "@codemirror/autocomplete";

/**
 * Creates a wikilink autocompletion extension.
 * `getNotes` is called each time the completion panel opens to get the latest list.
 */
export function wikilinkCompletion(
  getNotes: () => { path: string; title: string }[],
) {
  function source(ctx: CompletionContext): CompletionResult | null {
    // Look backwards from cursor for "[[" that hasn't been closed.
    const line = ctx.state.doc.lineAt(ctx.pos);
    const textBefore = line.text.slice(0, ctx.pos - line.from);
    const idx = textBefore.lastIndexOf("[[");
    if (idx < 0) return null;
    // Make sure there's no closing "]]" between [[ and cursor.
    const after = textBefore.slice(idx + 2);
    if (after.includes("]]")) return null;

    const from = line.from + idx + 2;
    const filter = after.toLowerCase();

    const notes = getNotes();
    const options: Completion[] = notes
      .filter(
        (n) =>
          n.path.toLowerCase().includes(filter) ||
          n.title.toLowerCase().includes(filter),
      )
      .slice(0, 30)
      .map((n) => ({
        label: n.path.replace(/\.md$/, ""),
        detail: n.title,
        apply: (view, completion, fromPos, toPos) => {
          // Replace the partial text with "target]]"
          const insert = `${completion.label}]]`;
          view.dispatch({
            changes: { from: fromPos, to: toPos, insert },
            selection: { anchor: fromPos + insert.length },
          });
        },
      }));

    return { from, options, filter: false };
  }

  return autocompletion({
    override: [source],
    activateOnTyping: true,
  });
}
