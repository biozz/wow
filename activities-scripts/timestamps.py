# /// script
# dependencies = [
#   "typer==0.17.3",
#   "pyyaml==6.0.2",
# ]
# ///
# 
# This script converts filenames in certain directories into timestamps, formatted as `YYYYMMDDHHmmSS.md`.
# 
# It does so by following these rules in order:
# - if the filename contains at least a date, either formatted as `YYYYMMDD` or `YYYY_MM_DD` it coverts to it to `YYYYMMDDHHmmSS.md`, settings the time to 120000'
# - otherwise inspects YAML frontmatter of the file and uses the `created` field
# - otherwise outputs the filename as is, asking the user to enter the timestamp manually
# 
# The scripts also uses external CLI utility `fd` to find if there are any filename conflicts.
# In case of conflicts, the script increments the timestamp by 1 second until it finds a free filename.
# The same applies for user generated timestamp input.
# The directory, which contains target files is passed as a command line argument, separately from the `notes_dir`, because the script is intended to change only a subset of files.
# 
# At the time of writing, the `notes_dir` is structured as `inbox/activities/{content,analysis,etc.}/<files>`.
# 
# Here is also an example YAML frontmatter:
# ---
# # other fields
# created: 2024-03-12T12:16:13
# ---
#
# Usage:
# uv run timestamps.py --dry-run ~/notes ~/notes/inbox/activities/content
# uv run timestamps.py ~/notes ~/notes/inbox/activities/content
import typer
import re
import yaml
import subprocess
from pathlib import Path
import datetime as dt

def extract_date_from_filename(filename: str) -> dt.datetime | None:
    """Extract date from filename using YYYYMMDD or YYYY_MM_DD patterns."""
    # Remove file extension
    name_without_ext = Path(filename).stem
    
    # Try YYYYMMDD pattern
    match = re.search(r'(\d{8})', name_without_ext)
    if match:
        date_str = match.group(1)
        try:
            return dt.datetime.strptime(date_str, '%Y%m%d')
        except ValueError:
            pass
    
    # Try YYYY_MM_DD pattern
    match = re.search(r'(\d{4})_(\d{2})_(\d{2})', name_without_ext)
    if match:
        year, month, day = match.groups()
        try:
            return dt.datetime(int(year), int(month), int(day))
        except ValueError:
            pass
    
    return None


def extract_created_from_frontmatter(file_path: Path) -> tuple[dt.datetime | None, str | None]:
    """Extract created timestamp from YAML frontmatter.
    
    Returns:
        Tuple of (datetime or None, error message or None)
    """
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Check if file has frontmatter
    if not content.startswith('---'):
        return None, "File does not start with frontmatter delimiter (---)"
    
    # Extract frontmatter
    parts = content.split('---', 2)
    if len(parts) < 3:
        return None, "Invalid frontmatter format: missing closing delimiter"
    
    frontmatter = parts[1].strip()
    if not frontmatter:
        return None, "Empty frontmatter section"
    
    # Parse YAML
    try:
        yaml_data = yaml.safe_load(frontmatter)
    except yaml.YAMLError as e:
        return None, f"Invalid YAML in frontmatter: {e}"
    
    if not yaml_data:
        return None, "Frontmatter is empty or contains no valid YAML"
    
    if 'created' not in yaml_data:
        return None, "No 'created' field found in frontmatter"
    
    created_value = yaml_data['created']
    
    # Check if PyYAML already parsed it as a datetime
    if isinstance(created_value, dt.datetime):
        return created_value, None
    
    # If it's not a string, try to convert it
    if not isinstance(created_value, str):
        return None, f"'created' field is not a string or datetime (got {type(created_value).__name__})"
    
    # Try different datetime formats
    formats = [
        '%Y-%m-%dT%H:%M:%S',
        '%Y-%m-%dT%H:%M:%SZ',
        '%Y-%m-%d %H:%M:%S',
        '%Y-%m-%d'
    ]
    
    for fmt in formats:
        try:
            return dt.datetime.strptime(created_value, fmt), None
        except ValueError:
            continue
    
    return None, f"Could not parse 'created' value '{created_value}' with any supported format"

def check_filename_conflict(target_dir: Path, new_filename: str) -> bool:
    """Check if filename already exists using fd command."""
    result = subprocess.run(
        ['fd', new_filename],
        cwd=str(target_dir),
        capture_output=True,
        text=True,
        check=False
    )
    return bool(result.stdout.strip())

def find_free_filename(target_dir: Path, base_timestamp: dt.datetime) -> str:
    """Find a free filename by incrementing timestamp if needed."""
    timestamp = base_timestamp
    while True:
        filename = f"{timestamp.strftime('%Y%m%d%H%M%S')}.md"
        if not check_filename_conflict(target_dir, filename):
            return filename
        # Increment by 1 second
        timestamp = timestamp + dt.timedelta(seconds=1)

def get_manual_timestamp() -> dt.datetime | None:
    """Get timestamp from user input."""
    while True:
        user_input = typer.prompt(
            "Enter timestamp manually (YYYYMMDDHHmmSS or YYYY-MM-DD HH:MM:SS or 'skip')",
            default="skip"
        )
        
        if user_input.lower() == 'skip':
            return None
        
        # Try YYYYMMDDHHmmSS format
        try:
            return dt.datetime.strptime(user_input, '%Y%m%d%H%M%S')
        except ValueError:
            pass
        
        # Try YYYY-MM-DD HH:MM:SS format
        try:
            return dt.datetime.strptime(user_input, '%Y-%m-%d %H:%M:%S')
        except ValueError:
            pass
        
        # Try YYYY-MM-DD format
        try:
            return dt.datetime.strptime(user_input, '%Y-%m-%d')
        except ValueError:
            pass
        
        typer.echo("Invalid timestamp format. Please use YYYYMMDDHHmmSS, YYYY-MM-DD HH:MM:SS, or YYYY-MM-DD")


class Processor:
    def __init__(self, notes_path: Path, target_path: Path, dry_run: bool):
        self.notes_path = notes_path
        self.target_path = target_path
        self.dry_run = dry_run
        self.md_files = list(target_path.glob("*.md"))
        
    def should_skip_file(self, file_path: Path) -> bool:
        """Check if file should be skipped (already in timestamp format)."""
        if Path(file_path.name).stem.isdigit():
            typer.echo(f"  Skipping {file_path.name} (already in timestamp format)")
            return True
        return False
    
    def get_timestamp_from_filename(self, file_path: Path) -> dt.datetime | None:
        """Try to extract timestamp from filename with default time."""
        date_from_filename = extract_date_from_filename(file_path.name)
        if not date_from_filename:
            return None
        return date_from_filename.replace(hour=12, minute=0, second=0)
    
    def get_timestamp_from_frontmatter(self, file_path: Path) -> tuple[dt.datetime | None, str | None]:
        """Try to extract timestamp from frontmatter."""
        return extract_created_from_frontmatter(file_path)
    
    def get_manual_timestamp_interactive(self, file_path: Path, error_msg: str | None) -> dt.datetime | None:
        """Get timestamp from user input during interactive mode."""
        if error_msg:
            typer.echo(f"  No date found in frontmatter: {error_msg}")
        else:
            typer.echo("  No date found in filename or frontmatter")
        
        manual_timestamp = get_manual_timestamp()
        if manual_timestamp:
            typer.echo(f"  Manual timestamp: {manual_timestamp.strftime('%Y-%m-%d %H:%M:%S')}")
        else:
            typer.echo(f"  Skipping {file_path.name}")
        return manual_timestamp
    
    def resolve_timestamp(self, file_path: Path) -> dt.datetime | None:
        """Resolve timestamp for a file using all available methods."""
        # Try filename first
        timestamp = self.get_timestamp_from_filename(file_path)
        if timestamp:
            typer.echo(f"  Date from filename: {timestamp.strftime('%Y-%m-%d')} (set to 12:00:00)")
            return timestamp
        
        # Try frontmatter
        timestamp, error_msg = self.get_timestamp_from_frontmatter(file_path)
        if timestamp:
            typer.echo(f"  Date from frontmatter: {timestamp.strftime('%Y-%m-%d %H:%M:%S')}")
            return timestamp
        
        # In dry run mode, just report what would happen
        if self.dry_run:
            if error_msg:
                typer.echo(f"{file_path.relative_to(self.notes_path)} -> [SKIP - {error_msg}]")
            else:
                typer.echo(f"{file_path.relative_to(self.notes_path)} -> [SKIP - no date found]")
            return None
        
        # Interactive mode - ask user
        return self.get_manual_timestamp_interactive(file_path, error_msg)
    
    def process_file(self, file_path: Path) -> bool:
        """Process a single file. Returns True if processed, False if skipped."""
        typer.echo(f"Processing {file_path.name}...")
        
        if self.should_skip_file(file_path):
            return False
        
        timestamp = self.resolve_timestamp(file_path)
        if not timestamp:
            return False
        
        new_filename = find_free_filename(self.notes_path, timestamp)
        
        if self.dry_run:
            typer.echo(f"{file_path.relative_to(self.notes_path)} -> {self.target_path.relative_to(self.notes_path) / new_filename}")
            return True
        
        return self.rename_file(file_path, new_filename)
    
    def rename_file(self, file_path: Path, new_filename: str) -> bool:
        """Rename the file and handle errors."""
        new_path = self.target_path / new_filename
        try:
            file_path.rename(new_path)
            typer.echo(f"  ✓ Renamed to {new_filename}")
            return True
        except Exception as e:
            typer.echo(f"  ✗ Error renaming: {e}")
            return False
    
    def process_all_files(self) -> None:
        """Process all markdown files in the target directory."""
        typer.echo(f"Found {len(self.md_files)} markdown files")
        
        if self.dry_run:
            typer.echo("DRY RUN MODE - No files will be changed")
        
        for file_path in self.md_files:
            self.process_file(file_path)


def validate_directories(notes_dir: str, target_dir: str) -> tuple[Path, Path]:
    """Validate that directories exist and are valid."""
    notes_path = Path(notes_dir)
    target_path = Path(target_dir)
    
    if not notes_path.exists():
        typer.echo(f"Error: Notes directory '{notes_dir}' does not exist")
        raise typer.Exit(1)
    
    if not notes_path.is_dir():
        typer.echo(f"Error: '{notes_dir}' is not a directory")
        raise typer.Exit(1)
    
    if not target_path.exists():
        typer.echo(f"Error: Directory '{target_dir}' does not exist")
        raise typer.Exit(1)
    
    if not target_path.is_dir():
        typer.echo(f"Error: '{target_dir}' is not a directory")
        raise typer.Exit(1)
    
    return notes_path, target_path


def main(notes_dir: str = typer.Argument(..., help="Root directory for notes"), 
         target_dir: str = typer.Argument(..., help="Directory containing files to rename"),
         dry_run: bool = typer.Option(False, "--dry-run", help="Show what would be changed without making changes")):
    """Convert filenames to timestamps based on date patterns or frontmatter."""
    notes_path, target_path = validate_directories(notes_dir, target_dir)
    Processor(notes_path, target_path, dry_run).process_all_files()

if __name__ == "__main__":
    typer.run(main)