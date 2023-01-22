#!/usr/bin/python3

"""Script to list plugin modules."""

import argparse
import json
import os


def args() -> argparse.Namespace:
    """ Reads the stdin arguments.

        Returns: arguments namespace.
    """
    args = argparse.ArgumentParser(
        description="Script to list plugin modules and prints them as JSON-encoded array string to stdout",
    )
    args.add_argument("-p", "--path", help="Base directory to list.", required=True, type=str)
    o = args.parse_args()
    return o


def main(path: str) -> None:
    """ Script entrypoint.

    It prints JSON-encoded array of plugins.

    Args:
        path: Base dir to list.
    """
    plugins = os.listdir(path)
    print(json.dumps(plugins))


if __name__ == "__main__":
    main(args().path)
