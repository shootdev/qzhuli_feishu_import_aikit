"""Run the Python app setup shipped with the skill."""
from pathlib import Path
import runpy
import sys

if __name__ == "__main__":
    scripts = Path(__file__).resolve().parent / "skills" / "feishu-enterprise-app-setup" / "scripts"
    sys.path.insert(0, str(scripts))
    runpy.run_path(str(scripts / "feishu_app.py"), run_name="__main__")
