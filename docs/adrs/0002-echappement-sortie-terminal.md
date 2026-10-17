<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# ADR-0002 : Échappement de toute sortie terminal

## Statut

Acceptée, 2026-10-17.

## Contexte

klock affiche des textes venus du serveur : noms, messages d'Event, labels,
en-têtes. Il affiche aussi la valeur de `-o` dans ses erreurs. Un tenant peut
écrire une séquence OSC 52 dans le message d'un Event. Quand l'admin lance
`klock events`, son presse-papiers change (constat C1 de `cso-1.md`).

Le mode custom-columns échappait déjà ses cellules. Le mode table ne le
faisait pas : plusieurs branches de `parseCell` renvoient la cellule telle
quelle. Les suggestions du filtre (N1) et le namespace restaient bruts aussi.

## Décision

klock passe chaque texte issu du serveur ou de `-o` par une seule fonction,
`sanitizeCell`, puis borne chaque cellule avec `boundCell`.

1. `sanitizeCell` applique `printers.EscapeTerminal` de kubectl. Elle
   remplace ensuite par une séquence ASCII les contrôles, les caractères
   bidi, Zl, Zp, Cf et les `Other_Default_Ignorable_Code_Point`. U+202E,
   par exemple, devient une séquence ASCII lisible. Les sélecteurs de
   variante restent intacts, pour que les emoji s'affichent.
2. Le mode table l'applique une fois, à l'entrée : cellules, en-têtes,
   colonnes `-L`, namespace et suggestions du filtre. Le mode custom-columns
   l'applique à ses valeurs, à ses en-têtes et à ses messages `<error: …>`.
   Les messages d'erreur de `-o` passent aussi par elle, sans `%q`.
3. `boundCell` limite chaque cellule à 1024 points de code, suivis de `…`,
   dans les deux modes.

La clé de tri garde la valeur brute : elle n'est jamais affichée.

Alternatives écartées :

- Échapper dans `colorCell` seulement. Les branches qui renvoient la cellule
  brute, comme la durée d'un Job ou `LAST SEEN`, resteraient ouvertes.
- Garder le seul `EscapeTerminal` de kubectl. Il laisse passer le C1, les
  contrôles bidi et les caractères invisibles.
- Utiliser `strconv.QuoteRune`. Il laisse brutes les runes ignorables que Go
  tient pour imprimables, comme U+3164 et U+115F.
- Échapper les sélecteurs de variante. Le texte d'un emoji casserait.

## Conséquences

Effets positifs :

- Une seule fonction couvre les deux modes. Un nouveau chemin d'affichage
  doit passer par elle.
- Les caractères invisibles et bidi deviennent visibles dans la cellule. Une
  valeur ne peut plus usurper l'affichage.
- Une valeur trop longue ne remplit plus l'écran. Le `…` signale la coupe.

Effets négatifs :

- Un texte coréen qui utilise U+3164 s'affiche échappé. Le compromis est
  accepté.
- La fin d'une valeur de plus de 1024 points de code n'est plus visible. Le
  mode custom-columns permet de cibler une sous-partie.

Points ouverts, hors de ce change :

- `Table.SetError` n'est pas échappé (cso-2, O1, Low).
- La suggestion n'est pas bornée (cso-2, O2, Info), ouvert à la date de
  `cso-2.md`.
- `sanitizeCell` traite la chaîne entière avant la borne (cso-2, N3, Info).
- La pagination de la liste initiale reste hors périmètre.
