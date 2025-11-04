"""SQLAlchemy 基类"""

from sqlalchemy.orm import DeclarativeBase


class Base(DeclarativeBase):
    """项目中所有 ORM 模型的基类"""

    pass
