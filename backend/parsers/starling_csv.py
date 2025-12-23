"""Starling Bank CSV statement parser."""
import csv
from datetime import datetime, date
from io import StringIO


class StarlingCSVParser:
    """Parser for Starling Bank CSV exports.

    CSV Format:
    Date,Counter Party,Reference,Type,Amount (GBP),Balance (GBP),Spending Category,Notes
    01/11/2025,Tesco,TESCO METRO,APPLE PAY,-7.10,18.06,GROCERIES,
    """

    # Map Starling categories to our category names
    CATEGORY_MAP = {
        "GROCERIES": "Groceries",
        "EATING_OUT": "Food & Dining",
        "ENTERTAINMENT": "Entertainment",
        "TRANSPORT": "Transport",
        "SHOPPING": "Shopping",
        "BILLS": "Utilities",
        "INCOME": "Income",
        "TRANSFERS": "Transfer In",
        "GENERAL": "Other",
        "LIFESTYLE": "Personal Care",
        "HOLIDAYS": "Travel",
        "FAMILY": "Other",
        "CHARITY": "Other",
        "GAMBLING": "Entertainment",
        "SAVINGS": "Transfer Out",
        "PAYMENTS": "Transfer Out",
    }

    def parse(self, content: bytes) -> dict:
        """
        Parse Starling CSV statement.

        Args:
            content: CSV file bytes

        Returns:
            Dict with keys:
                - period_start: date
                - period_end: date
                - opening_balance: float | None
                - closing_balance: float | None
                - transactions: list[dict]
                - raw_text: str
        """
        # Decode CSV
        text = content.decode("utf-8-sig")  # Handle BOM if present
        reader = csv.DictReader(StringIO(text))

        transactions = []
        dates = []
        balances = []

        for row in reader:
            try:
                # Parse date (DD/MM/YYYY)
                txn_date = datetime.strptime(row["Date"], "%d/%m/%Y").date()
                dates.append(txn_date)

                # Parse amount
                amount = float(row["Amount (GBP)"].replace(",", ""))

                # Parse balance
                balance = float(row["Balance (GBP)"].replace(",", ""))
                balances.append(balance)

                # Build description
                counter_party = row.get("Counter Party", "").strip()
                reference = row.get("Reference", "").strip()
                txn_type = row.get("Type", "").strip()

                if reference and reference != counter_party:
                    description = f"{counter_party} - {reference}"
                else:
                    description = counter_party

                if txn_type:
                    description = f"{description} ({txn_type})"

                # Get Starling category
                starling_category = row.get("Spending Category", "").strip()
                mapped_category = self.CATEGORY_MAP.get(starling_category)

                transactions.append({
                    "date": txn_date,
                    "description": description,
                    "amount": amount,
                    "balance": balance,
                    "starling_category": starling_category,
                    "mapped_category": mapped_category,
                    "notes": row.get("Notes", "").strip() or None,
                })

            except (KeyError, ValueError) as e:
                # Skip malformed rows
                continue

        # Determine period and balances
        if dates:
            period_start = min(dates)
            period_end = max(dates)
        else:
            period_start = period_end = date.today()

        # Opening balance: first transaction's balance - first transaction's amount
        # Closing balance: last transaction's balance
        opening_balance = None
        closing_balance = None

        if transactions:
            # Sort by date while keeping original order for same-day items
            transactions = [
                txn for _, txn in sorted(
                    enumerate(transactions),
                    key=lambda pair: (pair[1]["date"], pair[0]),
                )
            ]

            # Opening = balance before first transaction
            first_txn = transactions[0]
            opening_balance = first_txn["balance"] - first_txn["amount"]

            # Closing = last balance
            closing_balance = transactions[-1]["balance"]

        return {
            "period_start": period_start,
            "period_end": period_end,
            "opening_balance": opening_balance,
            "closing_balance": closing_balance,
            "transactions": transactions,
            "raw_text": text,
        }
