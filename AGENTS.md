- Follow the applicable [jstack RFCs](https://github.com/jmcampanini/jstack/tree/main/rfcs), available locally under `~/.jstack/rfcs/`.
- Treat command help as the canonical user-facing documentation.
- Keep the README as a landing page.
- Keep command help and the README consistent with observable behavior.
- Run repository tasks with `mise run <task>` instead of invoking pinned tools directly.
- Use `mise run check` for local and CI verification.
- Apply formatting with `mise run fmt`.
- Apply dependency cleanup with `mise run tidy`.
- Upgrade a tool pin with this sequence:
  1. Run `mise upgrade --bump <tool>`.
  2. Run `mise lock`.
  3. Run `mise run check`.
- Update the handwritten `output-formats` help topic in `internal/cli/output_formats.go` in the same change as any record schema it describes.
- Use `plan`, `assigned`, `available`, `ready to assign`, and `target` in user-facing text instead of the API's older terminology.
