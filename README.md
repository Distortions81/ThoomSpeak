# goThoom

A friendly open-source client for the classic **Clan Lord** game.

## Download and Play

1. Download the latest release for Windows, macOS or Linux.
2. Unzip the file.
3. Run the program:
   - **Windows/macOS:** open the app.
   - **Linux:** run `./gothoom` (set it executable if needed).

On the first start the game fetches missing images and sounds into a `data` folder.

## Customizing

- Put a `background.png` or `splash.png` in the `data` folder to change the look.
- Text‑to‑speech voices go in `data/piper/voices`. The script `scripts/download_piper.sh` can download them for you.

## Extras

- Optional plugins can add new features. Place `.go` files in a `plugins` folder next to the program or inside `data/plugins/` and restart the game.
- Advanced flags and building from source are meant for developers and are not covered here.

## Troubleshooting

- If assets fail to download, remove the partial files in `data/` and restart.
- Linux users may need OpenGL and X11 libraries installed.
- For help or issues, please open an issue on GitHub.

## License

MIT. Game assets and "Clan Lord" remain the property of their owners.
