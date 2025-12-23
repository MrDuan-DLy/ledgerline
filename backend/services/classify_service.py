"""Classification service - rule-based transaction categorization."""
import re
from sqlalchemy.orm import Session

from ..models import Rule, Category, Transaction


class ClassifyService:
    """Handles automatic transaction classification."""

    def __init__(self, db: Session):
        self.db = db
        self._rules_cache: list[Rule] | None = None

    def _get_active_rules(self) -> list[Rule]:
        """Get all active rules, ordered by priority (highest first)."""
        if self._rules_cache is None:
            self._rules_cache = (
                self.db.query(Rule)
                .filter(Rule.is_active == True)
                .order_by(Rule.priority.desc())
                .all()
            )
        return self._rules_cache

    def invalidate_cache(self):
        """Clear rules cache after rule changes."""
        self._rules_cache = None

    def classify(self, description: str) -> tuple[int | None, str]:
        """
        Classify a transaction description.

        Returns:
            (category_id, source) where source is 'rule' or 'unclassified'
        """
        rules = self._get_active_rules()
        description_upper = description.upper()

        for rule in rules:
            if self._matches(rule, description_upper):
                return (rule.category_id, "rule")

        return (None, "unclassified")

    def _matches(self, rule: Rule, description: str) -> bool:
        """Check if a rule matches the description."""
        pattern = rule.pattern.upper()

        if rule.pattern_type == "exact":
            return description == pattern
        elif rule.pattern_type == "contains":
            return pattern in description
        elif rule.pattern_type == "regex":
            try:
                return bool(re.search(rule.pattern, description, re.IGNORECASE))
            except re.error:
                return False
        return False

    def classify_transaction(self, transaction: Transaction) -> bool:
        """
        Apply classification to a transaction if not manually classified.

        Returns:
            True if classification was applied/changed
        """
        # Don't override manual classifications
        if transaction.category_source == "manual":
            return False

        category_id, source = self.classify(transaction.raw_description)

        if category_id != transaction.category_id:
            transaction.category_id = category_id
            transaction.category_source = source
            return True

        return False

    def reclassify_all(self) -> int:
        """
        Re-run classification on all non-manual transactions.

        Returns:
            Number of transactions updated
        """
        self.invalidate_cache()

        transactions = (
            self.db.query(Transaction)
            .filter(Transaction.category_source != "manual")
            .all()
        )

        updated = 0
        for txn in transactions:
            if self.classify_transaction(txn):
                updated += 1

        self.db.commit()
        return updated
