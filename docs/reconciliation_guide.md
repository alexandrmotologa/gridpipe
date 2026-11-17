# Cross-Tab Reconciliation Guide

This guide covers operational reconciliation workflows for financial, logistics, and procurement processes using gridpipe.

## Typical Reconciliation Scenarios

1. Budget vs. Actuals: Comparing planned expense line items against realized bank or invoice statements.
2. Supplier Invoices vs. Purchase Orders: Verifying that vendor billed quantities and unit prices match procurement agreements.
3. Regional Drop Consolidation: Aligning weekly sales sheets sent by independent retail branches against central ERP inventories.

## Variance Formula and Delta Calculations

For each row aligned by a primary key (e.g. SKU or Cost Code):

$$\Delta = V_{\text{actual}} - V_{\text{baseline}}$$

$$\text{Variance \%} = \frac{\Delta}{|V_{\text{baseline}}|} \times 100\%$$

If $|\Delta| \le \epsilon$ (where default $\epsilon = 0.0001$), the line item is classified as `MATCH`.
If $|\Delta| > \epsilon$, the line item is classified as `VARIANCE` with an explanatory audit note.

## CLI Execution

```bash
gridpipe reconcile \
  -i financial_close.xlsx \
  --sheet-a "Budget_Q1" \
  --sheet-b "Actuals_Q1" \
  -k 0 \
  -v 1 \
  --epsilon 0.01 \
  -o reconciled_report.xlsx
```

The resulting `reconciled_report.xlsx` contains:
1. `Reconciliation_Audit` worksheet with side-by-side values, dynamic `=C2-B2` formulas, percentage calculations, status badges, and audit commentary.
2. Automated color coding: Soft green for matches, soft red for discrepancies.
3. Summary totals row calculating aggregate net delta and baseline totals.
