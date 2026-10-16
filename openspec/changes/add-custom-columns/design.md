<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Design

Taille S : contraintes, solution, risques. Motivation dans `proposal.md`,
comportement attendu dans `specs/custom-columns-output/spec.md`.

## Context

- Toutes les requêtes de klock portent `Accept: as=Table`
  (`transformRequests`, `pkg/klock/klock.go`). Avec ce format, le serveur ne
  renvoie que les métadonnées de chaque objet : `.spec` et `.status` sont
  absents.
- `Printer.PrintObj` décode toujours une `Table` (`decodeIntoTable`) et
  échoue sur un objet ordinaire.
- `k8s.io/kubectl/pkg/cmd/get` fournit déjà le parsing des options :
  `NewCustomColumnsPrintFlags().ToPrinter(format)` reconnaît les deux
  préfixes, lit le fichier et renvoie un `*CustomColumnsPrinter` dont le
  champ `Columns` contient les en-têtes et les expressions normalisées.
  L'évaluation par objet (`printOneObject`) n'est pas exportée.

## Goals / Non-Goals

**Goals :** réutiliser le parsing de kubectl tel quel, pour garder ses
messages d'erreur ; parser chaque expression une seule fois, au démarrage ;
laisser le mode tableau intact.

**Non-Goals :** rendre `Printer` générique (une interface par mode
d'affichage) ; ajouter un journal à klock.

## Decisions

1. **Parsing.** `ParseCustomColumns(output)` appelle
   `get.NewCustomColumnsPrintFlags().ToPrinter(output)`. Une
   `NoCompatiblePrinterError` signale un autre format : la fonction renvoie
   alors `nil, nil`. Sinon elle parse chaque `FieldSpec` avec
   `jsonpath.New(header).AllowMissingKeys(true)` et préfixe une erreur de
   parsing par `column "<header>":` et une espace. Alternative écartée :
   réécrire le découpage de la spec et la lecture du fichier, soit une
   trentaine de lignes en double et des messages qui divergent de kubectl.
2. **Validation.** `Options.Validate` renvoie désormais les colonnes avec
   l'erreur. `Execute` parse donc une seule fois. Validate refuse aussi `-L`
   en mode custom, avec le message de kubectl. Le message des formats
   refusés liste `wide, custom-columns=..., custom-columns-file=...`.
3. **Requête.** `acceptHeader(customColumns bool) string` renvoie l'en-tête
   `Table` actuel, ou `""` en mode custom. La closure passée à
   `TransformRequests` ne pose l'en-tête que s'il n'est pas vide. Le serveur
   répond alors en JSON avec les objets complets, comme pour
   `kubectl get -o custom-columns`.
4. **Printer.** Nouveau champ `CustomColumns []CustomColumn`. S'il n'est pas
   vide, `PrintObj` prend le chemin custom : `SetHeaders` avec les
   en-têtes, puis `customRow(obj, eventType)` construit la ligne et
   `AddRow` l'ajoute. L'extraction de `uid`, `name`, `namespace` et
   `creationTimestamp` sort de `addObjectToTable` dans `baseRow`, partagé par
   les deux chemins. `HasLeadingNamespaceColumn` reste à `false` en mode
   custom.
5. **Cellule.** `evalCell(path, obj)` reproduit `printOneObject` : valeurs
   jointes par `,`, `<none>` si le résultat est vide, `printers.EscapeTerminal`
   sur chaque valeur, `<error: …>` si l'évaluation échoue. La coloration
   réutilise la branche par défaut de `parseCell` (`FractionStyle` puis
   `StatusColumn`), sauf pour un événement `DELETED`.

Interfaces publiques ajoutées ou modifiées, paquet `pkg/klock` :

```go
// CustomColumn is one user-defined column: its header and parsed JSONPath.
type CustomColumn struct {
    Header string
    Path   *jsonpath.JSONPath
}

// ParseCustomColumns parses -o custom-columns=… or -o custom-columns-file=….
// It returns nil, nil when output selects another format.
func ParseCustomColumns(output string) ([]CustomColumn, error)

// Validate checks the options and returns the parsed custom columns, if any.
func (o Options) Validate() ([]CustomColumn, error)

// Printer gains: CustomColumns []CustomColumn
```

Fonctions internes testables : `acceptHeader(bool) string`,
`evalCell(*jsonpath.JSONPath, map[string]any) string`,
`(*Printer).customRow(*unstructured.Unstructured, watch.EventType)
(table.Row, error)`.

## Journalisation

klock n'a pas de journal. Les erreurs remontent par `Table.SetError` et
déclenchent le redémarrage du watch. Un seul niveau suffit : le changement
n'ajoute aucun traitement en arrière-plan qui demanderait des traces. Les
options invalides arrêtent klock avec `ERROR: <message>` sur la sortie
standard (comportement actuel de `cmd.InitAndExecute`). Une erreur
d'évaluation reste visible dans la cellule, `<error: …>`, sans couper le
watch. Aucun chemin ne l'ignore en silence.

## Risks / Trade-offs

- [Objets complets plus lourds que la table du serveur] → Seul le mode
  custom les demande, comme kubectl.
- [`kubectl` peut changer `ToPrinter` ou `Column`] → API stable depuis des
  années. Les tests de parsing casseront à la montée de version.
- [Le message de fichier introuvable de kubectl finit par `\n`] → klock
  retire les blancs finaux du message avant de l'afficher.
