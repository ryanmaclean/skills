#!/usr/bin/env nu
# nb-append.nu: append stdin text to the last markdown cell of a Datadog notebook via the pup CLI.
# Ported from the retired dd-port-work/nb_append.py.
# Usage: printf "text" | nu nb-append.nu <notebook-id>

def main [nb_id: string] {
  let text = (open --raw /dev/stdin | decode utf-8 | str trim --right --char "\n")
  let nb = (^pup notebooks get $nb_id --output json | from json)
  mut attrs = $nb.data.attributes
  for k in [author created deleted modified status] { $attrs = ($attrs | reject -o $k) }
  let cells = $attrs.cells
  let n = ($cells | length)
  if $n == 0 { error make {msg: "notebook has no cells"} }
  let idx = (($cells | enumerate | where {|c| $c.item.attributes.definition.type? == "markdown" } | last).index)
  let old = ($cells | get $idx | get attributes.definition.text)
  $attrs.cells = ($cells | update $idx {|c| $c | update attributes.definition.text $"(($old | str trim --right --char "\n"))\n\n($text)\n" })
  let f = (mktemp -t nb-update.XXXXXX)
  {data: {type: "notebooks", id: $nb_id, attributes: $attrs}} | to json | save -f $f
  ^pup notebooks update $nb_id --file $f
  rm -f $f
}
