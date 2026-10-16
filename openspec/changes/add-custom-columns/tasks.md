<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Tasks

## 0. Revue du plan

- [x] 0.1 plan-rev : taille S, accord de l'utilisateur sur proposal, specs,
  design et tasks ; vérifié par sa réponse dans la session

## 1. Parsing et validation des options

Interfaces (`pkg/klock`) : `type CustomColumn struct { Header string; Path
*jsonpath.JSONPath }`, `func ParseCustomColumns(output string)
([]CustomColumn, error)`, `func (o Options) Validate() ([]CustomColumn,
error)`.

- [ ] 1.1 `ParseCustomColumns` via `get.NewCustomColumnsPrintFlags()`,
  parsing JSONPath au démarrage, blancs finaux retirés des messages ;
  scénarios « Deux colonnes inline » (en-têtes), « Expression entre
  accolades », « Fichier valide », « Spec vide », « Colonne sans
  expression », « Expression JSONPath invalide », « Fichier introuvable »,
  « En-têtes et expressions en nombre différent » ; vérifié par
  `go test ./pkg/klock`
- [ ] 1.2 `Options.Validate` renvoie les colonnes, refuse `-L` en mode
  custom, liste les nouveaux formats dans le message de refus ; scénarios
  « Combinaison avec -L », « Format toujours refusé » ; vérifié par
  `go test ./pkg/klock`

## 2. Requête au serveur

Interface interne : `func acceptHeader(customColumns bool) string`.

- [ ] 2.1 `acceptHeader` et closure `TransformRequests` qui n'envoie pas
  `as=Table` en mode custom ; scénarios « En-tête Accept en mode
  custom-columns », « En-tête Accept en mode tableau » ; vérifié par
  `go test ./pkg/klock`

## 3. Affichage des colonnes

Interfaces internes : `func evalCell(path *jsonpath.JSONPath, obj
map[string]any) string`, `func (p *Printer) customRow(obj
*unstructured.Unstructured, eventType watch.EventType) (table.Row, error)`,
champ `Printer.CustomColumns []CustomColumn`.

- [ ] 3.1 `evalCell` : jointure `,`, `<none>`, `EscapeTerminal`,
  `<error: …>` ; scénarios « Champ absent », « Valeurs multiples », « Champ
  hors de la table du serveur », « Erreur d'évaluation sur un objet »,
  « Annotation avec séquence ANSI » ; vérifié par `go test ./pkg/klock`
- [ ] 3.2 `baseRow` extrait de `addObjectToTable`, `customRow` et chemin
  custom de `PrintObj` (en-têtes, coloration, suppression, pas de
  `NAMESPACE`) ; scénarios « Deux colonnes inline » (cellules),
  « Modification », « Suppression », « Fraction incomplète », « Statut
  d'erreur », « Ligne supprimée », « Toutes les namespaces » ; vérifié par
  `go test ./...`, tests du mode tableau toujours verts
- [ ] 3.3 `Execute` branche les colonnes sur `Printer` et la requête ;
  vérifié par `go build ./...` et `go vet ./...`

## 4. Ligne de commande et documentation

- [ ] 4.1 `cmd/root.go` : complétion de `-o` (`wide`, `custom-columns=`,
  `custom-columns-file=`), aide et exemples ; scénario « Complétion » ;
  vérifié par `go test ./cmd` et `go run . __complete pods -o ""`
- [ ] 4.2 README : section sur `-o custom-columns` et
  `-o custom-columns-file` ; vérifié par `npx markdownlint-cli2` et
  `uvx reuse lint` sans erreur sur les fichiers suivis

## 5. Vérification et livraison

- [ ] 5.1 verify : skeptic vérifie chaque scénario de la spec, verdict PASS
- [ ] 5.2 e2e : `/qa` sur un cluster `kind` jetable (jamais le contexte
  courant), pods réels, spec inline et fichier, création, modification,
  suppression
- [x] 5.3 ci — skipped: `test.yml` lance déjà `go test ./...`, aucun
  changement de pipeline nécessaire
- [ ] 5.4 review : judge lance `/review`, aucun constat Critical ou
  Important ouvert
- [ ] 5.5 security : `/cso`, aucun constat bloquant
- [ ] 5.6 adr : `docs/adrs/0001-custom-columns-objets-complets.md`, format
  Nygard
- [ ] 5.7 release-doc : writer lance `/document-release` sur les docs
  touchées par la branche

## Workflow follow-up

- archive : `openspec archive add-custom-columns --yes`, puis commit sur la
  branche.
- ship : `/ship` (remote `origin` présent).
