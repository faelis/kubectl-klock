<!--
SPDX-FileCopyrightText: 2026 faelis

SPDX-License-Identifier: CC-BY-4.0
-->

# Tasks

## 0. Revue du plan

- [x] 0.1 plan-rev : taille S, accord de l'utilisateur sur proposal, specs,
  design et tasks ; vérifié par sa réponse dans la session (« ok, validé,
  lance l'exécution »)

## 1. Échappement étendu

Interface interne (`pkg/klock`) : `func sanitizeCell(s string) string`,
inchangée.

- [x] 1.1 `sanitizeCell` échappe aussi `Other_Default_Ignorable_Code_Point`
  et écrit chaque rune en ASCII ; scénarios « Caractères ignorables par
  défaut », « Annotation avec séquence ANSI », « Saut de ligne et
  tabulation », « Autres caractères de contrôle », « Caractères
  invisibles » ; vérifié par `go test ./pkg/klock`

## 2. Mode table

Interfaces internes : `func (p *Printer) addObjectToTable(objTable
*metav1.Table, eventType watch.EventType) (tea.Cmd, error)`, `func (p
*Printer) updateColDefHeaders(objTable *metav1.Table)`, `func
labelColumnHeader(label string) string`, toutes inchangées.

- [x] 2.1 Cellules texte, liste et objet échappées avant `parseCell` ;
  scénarios « Message d'Event avec OSC 52 », « Saut de ligne dans une
  colonne de CR », « Caractère invisible en mode table », « Cellule
  liste », « Valeur numérique inchangée », « Statut toujours coloré » ;
  vérifié par `go test ./pkg/klock`
- [x] 2.2 En-têtes et colonnes `-L` échappés ; scénarios « Nom de colonne
  avec séquence ANSI », « Échappement après les majuscules », « Clé de
  label avec séquence ANSI », « Valeur de label avec séquence ANSI » ;
  vérifié par `go test ./pkg/klock`

## 3. Messages d'erreur de `-o`

Interfaces : `func ParseCustomColumns(output string) ([]CustomColumn,
error)`, `func (o Options) Validate() ([]CustomColumn, error)`,
inchangées.

- [x] 3.1 Erreurs de kubectl, du parseur JSONPath et refus de `-L`
  échappés ; scénarios « Spec sans deux-points échappée », « Expression
  invalide échappée », « Combinaison avec -L échappée » ; vérifié par
  `go test ./pkg/klock`

## 4. Longueur des cellules

Interface interne : `func boundCell(s string) string`, constante
`maxCellRunes = 1024`.

- [x] 4.1 `boundCell` appliquée au résultat de `evalCell` et aux cellules
  texte du mode table ; scénarios « Cellule de 2000 caractères »,
  « Cellule de 1024 caractères », « Borne après échappement », « Valeur de
  2000 caractères », « Valeurs jointes trop longues », « Cellule très
  longue » ; vérifié par `go test ./...`

## 4b. Corrections de revue

- [x] 4b.1 Cellule `<error: …>` bornée, `-o` refusé échappé sans `%q`,
  en-tête échappé une seule fois dans les erreurs, docstring d'`evalCell` ;
  scénarios « Message d'erreur borné », « Format refusé avec caractère
  ignorable », « En-tête échappé une seule fois dans les erreurs » ;
  vérifié par `go test ./pkg/klock`

## 4c. Corrections de sécurité

- [ ] 4c.1 Nom échappé dans la suggestion du filtre, namespace échappé
  dans la cellule `NAMESPACE` ; scénarios « Role au nom piégé »,
  « Suggestion en mode custom-columns », « Namespace piégé avec -A » ;
  vérifié par `go test ./pkg/klock`

## 5. Vérification et livraison

- [x] 5.1 verify : skeptic, chaque scénario du delta, verdict PASS ;
  vérifié par le rapport du skeptic (`VERDICT: PASS`, 0 Critical/Important)
- [x] 5.2 e2e : cluster kind jetable, Event dont le message contient une
  séquence OSC 52, `klock events` affiche `^[]52;…` ; contexte `dok-test`
  et `~/.kube/config` non touchés ; vérifié par le rapport e2e (11 étapes
  OK, `E2E: GREEN`, cluster supprimé)
- [x] 5.3 ci — skipped: aucun changement de pipeline
  (`.github/workflows/` inchangé)
- [x] 5.4 review : judge `/review`, PASS ; vérifié par `reviews/review-1.md`
  et `reviews/review-2.md` (`REVIEW: PASS`, 0 Critical/Important)
- [ ] 5.5 security : judge `/cso`, PASS
- [ ] 5.6 adr : `docs/adrs/0002-…`, format Nygard
- [ ] 5.7 release-doc : writer `/document-release` ; README, phrase sur
  l'échappement complétée (mode table, borne de 1024 caractères,
  sélecteurs de variante laissés intacts)

## Workflow follow-up

- archive : `openspec archive sanitize-table-output --yes`
- ship : push sur le fork, PR vers `applejag/kubectl-klock`
