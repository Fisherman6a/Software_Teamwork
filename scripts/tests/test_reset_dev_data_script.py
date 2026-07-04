import os
import shutil
import stat
import subprocess
import textwrap
import tempfile
import unittest
from pathlib import Path
from typing import Optional


class ResetDevDataScriptTests(unittest.TestCase):
    def test_reset_with_yes_stops_backend_and_removes_volumes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))

            result = self.run_reset(root, args=["--yes"])

            self.assertEqual(0, result.returncode, result.stderr)
            self.assertIn("local development data cleared", result.stdout)
            sequence = (root / "sequence.log").read_text(encoding="utf-8")
            self.assertIn("stop-backend\n", sequence)
            self.assertIn("docker:compose -f", sequence)
            self.assertIn(" config --quiet\n", sequence)
            self.assertIn(" down -v --remove-orphans\n", sequence)
            self.assertLess(sequence.index("stop-backend"), sequence.index(" down -v --remove-orphans"))

    def test_non_interactive_reset_requires_yes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))

            result = self.run_reset(root)

            self.assertEqual(2, result.returncode)
            self.assertIn("refusing to reset without --yes", result.stderr)
            sequence = (root / "sequence.log").read_text(encoding="utf-8")
            self.assertIn(" config --quiet\n", sequence)
            self.assertNotIn("stop-backend", sequence)
            self.assertNotIn(" down -v", sequence)

    def test_can_skip_backend_stop_and_keep_orphans(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = self.prepare_runtime(Path(directory))

            result = self.run_reset(root, args=["--yes", "--skip-stop-backend", "--keep-orphans"])

            self.assertEqual(0, result.returncode, result.stderr)
            sequence = (root / "sequence.log").read_text(encoding="utf-8")
            self.assertNotIn("stop-backend", sequence)
            self.assertIn(" down -v\n", sequence)
            self.assertNotIn("--remove-orphans", sequence)

    def prepare_runtime(self, root: Path) -> Path:
        script_source = Path.cwd() / "scripts" / "local" / "reset-dev-data.sh"
        script_target = root / "scripts" / "local" / "reset-dev-data.sh"
        script_target.parent.mkdir(parents=True)
        shutil.copy2(script_source, script_target)
        script_target.chmod(script_target.stat().st_mode | stat.S_IXUSR)

        loader_source = Path.cwd() / "scripts" / "config" / "load-profile.sh"
        loader_target = root / "scripts" / "config" / "load-profile.sh"
        loader_target.parent.mkdir(parents=True)
        shutil.copy2(loader_source, loader_target)
        loader_target.chmod(loader_target.stat().st_mode | stat.S_IXUSR)

        stop_target = root / "scripts" / "local" / "stop-backend.sh"
        self.write_executable(
            stop_target,
            """\
            #!/usr/bin/env bash
            echo "stop-backend" >> "$FAKE_SEQUENCE"
            """,
        )

        (root / "config" / "ctl").mkdir(parents=True)
        (root / "deploy").mkdir()
        (root / "deploy" / "docker-compose.yml").write_text("services: {}\n", encoding="utf-8")
        (root / ".env.local").write_text(
            textwrap.dedent(
                """\
                POSTGRES_ADMIN_URL=postgres://example/postgres
                """
            ),
            encoding="utf-8",
        )

        fake_bin = root / "fake-bin"
        fake_bin.mkdir()
        for command in ["bash", "cp", "dirname", "mkdir"]:
            target = shutil.which(command)
            if target is None:
                raise AssertionError(f"{command} is required to run reset-dev-data.sh tests")
            os.symlink(target, fake_bin / command)

        self.write_executable(
            fake_bin / "docker",
            """\
            #!/usr/bin/env bash
            echo "docker:$*" >> "$FAKE_SEQUENCE"
            exit 0
            """,
        )
        self.write_executable(
            fake_bin / "go",
            """\
            #!/usr/bin/env bash
            if [[ "$PWD" == "$FAKE_ROOT/config/ctl" && "$1" == "run" && "$2" == "." && "$3" == "render" ]]; then
              format="dotenv"
              out=""
              while (($# > 0)); do
                case "$1" in
                  --format)
                    format="$2"
                    shift 2
                    ;;
                  --out)
                    out="$2"
                    shift 2
                    ;;
                  *)
                    shift
                    ;;
                esac
              done
              [[ -n "$out" ]] || exit 64
              mkdir -p "$(dirname "$out")"
              if [[ "$format" == "shell" ]]; then
                while IFS= read -r line || [[ -n "$line" ]]; do
                  [[ -z "$line" || "$line" =~ ^[[:space:]]*# || "$line" != *=* ]] && continue
                  key="${line%%=*}"
                  value="${line#*=}"
                  printf "export %s=%q\\n" "$key" "$value"
                done < "$FAKE_ROOT/.env.local" > "$out"
              else
                cp "$FAKE_ROOT/.env.local" "$out"
              fi
              exit 0
            fi
            exit 0
            """,
        )
        return root

    def run_reset(
        self,
        root: Path,
        args: Optional[list[str]] = None,
    ) -> subprocess.CompletedProcess[str]:
        env = os.environ.copy()
        env["FAKE_ROOT"] = str(root)
        env["FAKE_SEQUENCE"] = str(root / "sequence.log")
        env["PATH"] = f"{root / 'fake-bin'}{os.pathsep}{env['PATH']}"
        return subprocess.run(
            [str(root / "scripts" / "local" / "reset-dev-data.sh"), *(args or [])],
            cwd=root,
            env=env,
            text=True,
            capture_output=True,
            check=False,
        )

    def write_executable(self, path: Path, content: str) -> None:
        path.write_text(textwrap.dedent(content), encoding="utf-8")
        path.chmod(path.stat().st_mode | stat.S_IXUSR)


if __name__ == "__main__":
    unittest.main()
