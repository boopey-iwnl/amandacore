using System.Diagnostics;
using System.Text;

namespace AmandaCore.LocalControls;

internal sealed class LocalControlsForm : Form
{
    private readonly string repoRoot;
    private readonly string devScriptsRoot;
    private readonly List<Button> commandButtons = new();
    private readonly TextBox outputBox = new();
    private readonly Label statusLabel = new();
    private readonly Label repoLabel = new();

    public LocalControlsForm()
    {
        repoRoot = ResolveRepoRoot();
        devScriptsRoot = Path.Combine(repoRoot, "Infra", "dev");

        Text = "Local Playable Slice Controls";
        StartPosition = FormStartPosition.CenterScreen;
        MinimumSize = new Size(960, 680);
        Size = new Size(1080, 760);
        Font = new Font("Segoe UI", 9F);

        BuildLayout();
        AppendOutput($"Repo root: {repoRoot}");
        AppendOutput("Ready.");
    }

    private void BuildLayout()
    {
        var root = new TableLayoutPanel
        {
            Dock = DockStyle.Fill,
            ColumnCount = 1,
            RowCount = 5,
            Padding = new Padding(16)
        };
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        root.RowStyles.Add(new RowStyle(SizeType.Percent, 100F));
        root.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        Controls.Add(root);

        var title = new Label
        {
            Text = "Local Playable Slice Controls",
            AutoSize = true,
            Font = new Font(Font.FontFamily, 14F, FontStyle.Bold),
            Margin = new Padding(0, 0, 0, 4)
        };
        root.Controls.Add(title, 0, 0);

        repoLabel.Text = repoRoot;
        repoLabel.AutoEllipsis = true;
        repoLabel.Dock = DockStyle.Fill;
        repoLabel.Margin = new Padding(0, 0, 0, 12);
        root.Controls.Add(repoLabel, 0, 1);

        var buttonPanel = new FlowLayoutPanel
        {
            AutoSize = true,
            Dock = DockStyle.Fill,
            WrapContents = true,
            Margin = new Padding(0, 0, 0, 12)
        };
        root.Controls.Add(buttonPanel, 0, 2);

        AddCommandButton(buttonPanel, "Start local stack", () =>
            RunPowerShellScriptAsync("Start local stack", Path.Combine(devScriptsRoot, "start-local.ps1"), "-BuildFirst"));

        AddCommandButton(buttonPanel, "Stop local stack", () =>
            RunPowerShellScriptAsync("Stop local stack", Path.Combine(devScriptsRoot, "stop-local.ps1")));

        AddCommandButton(buttonPanel, "Build local", () =>
            RunPowerShellScriptAsync("Build local", Path.Combine(devScriptsRoot, "build-local.ps1")));

        AddCommandButton(buttonPanel, "Build O3DE client", () =>
            RunPowerShellScriptAsync("Build O3DE client", Path.Combine(devScriptsRoot, "build-o3de-client.ps1")));

        AddCommandButton(buttonPanel, "Verify O3DE client", () =>
            RunPowerShellScriptAsync("Verify O3DE client", Path.Combine(devScriptsRoot, "verify-o3de-client.ps1")));

        AddCommandButton(buttonPanel, "Launch AmandaCore Launcher", LaunchAmandaCoreLauncherAsync);

        AddCommandButton(buttonPanel, "Open logs/output folder", OpenLogsFolderAsync);

        AddCommandButton(buttonPanel, "Collect diagnostics", () =>
            RunPowerShellScriptAsync("Collect diagnostics", Path.Combine(repoRoot, "Infra", "qa", "Collect-Diagnostics.ps1")));

        AddCommandButton(buttonPanel, "Open QA docs", () =>
            OpenFolderAsync("Open QA docs", Path.Combine(repoRoot, "Docs", "QA")));

        AddCommandButton(buttonPanel, "Reset test state", ResetTestStateAsync);

        AddCommandButton(buttonPanel, "Open admin portal", OpenAdminPortalAsync);

        outputBox.Dock = DockStyle.Fill;
        outputBox.Multiline = true;
        outputBox.ReadOnly = true;
        outputBox.ScrollBars = ScrollBars.Both;
        outputBox.WordWrap = false;
        outputBox.Font = new Font("Consolas", 9F);
        root.Controls.Add(outputBox, 0, 3);

        statusLabel.Text = "Ready.";
        statusLabel.AutoEllipsis = true;
        statusLabel.Dock = DockStyle.Fill;
        statusLabel.Font = new Font(Font.FontFamily, 9F, FontStyle.Bold);
        statusLabel.Margin = new Padding(0, 10, 0, 0);
        root.Controls.Add(statusLabel, 0, 4);
    }

    private void AddCommandButton(Control parent, string text, Func<Task> handler)
    {
        var button = new Button
        {
            Text = text,
            Width = 180,
            Height = 36,
            Margin = new Padding(0, 0, 10, 10),
            UseVisualStyleBackColor = true
        };
        button.Click += async (_, _) => await RunButtonCommandAsync(text, handler);
        commandButtons.Add(button);
        parent.Controls.Add(button);
    }

    private async Task RunButtonCommandAsync(string commandName, Func<Task> handler)
    {
        SetBusy(true, $"Running {commandName}...");
        try
        {
            await handler();
        }
        catch (Exception ex)
        {
            AppendOutput($"ERROR: {ex.Message}");
            SetStatus($"{commandName} failed.");
        }
        finally
        {
            SetBusy(false);
        }
    }

    private async Task RunPowerShellScriptAsync(string displayName, string scriptPath, params string[] scriptArguments)
    {
        if (!File.Exists(scriptPath))
        {
            AppendOutput($"{displayName} skipped. Script not found: {scriptPath}");
            SetStatus($"{displayName} skipped.");
            return;
        }

        var arguments = new List<string>
        {
            "-NoLogo",
            "-NoProfile",
            "-ExecutionPolicy",
            "Bypass",
            "-File",
            scriptPath
        };
        arguments.AddRange(scriptArguments);

        var exitCode = await RunProcessAsync("powershell.exe", arguments, repoRoot, displayName);
        AppendOutput($"Exit code: {exitCode}");
        SetStatus(exitCode == 0
            ? $"{displayName} finished with exit code 0."
            : $"{displayName} failed with exit code {exitCode}.");
    }

    private async Task LaunchAmandaCoreLauncherAsync()
    {
        var buildScript = Path.Combine(devScriptsRoot, "build-playable-client.ps1");
        var launcherExe = Path.Combine(
            repoRoot,
            "Client",
            "Launcher",
            "AmandaCore.Launcher",
            "bin",
            "Debug",
            "net8.0-windows",
            "AmandaCore.Launcher.exe");

        if (!File.Exists(buildScript))
        {
            AppendOutput($"Launcher build script not found: {buildScript}");
            SetStatus("Launch AmandaCore Launcher skipped.");
            return;
        }

        var command = string.Join("; ", new[]
        {
            "$ErrorActionPreference = 'Stop'",
            $"& {ToPowerShellString(buildScript)}",
            $"if (!(Test-Path {ToPowerShellString(launcherExe)})) {{ throw 'Launcher executable was not found.' }}",
            $"Start-Process -FilePath {ToPowerShellString(launcherExe)}",
            $"Write-Host ('Launcher opened: ' + {ToPowerShellString(launcherExe)})"
        });

        var exitCode = await RunProcessAsync(
            "powershell.exe",
            new[]
            {
                "-NoLogo",
                "-NoProfile",
                "-ExecutionPolicy",
                "Bypass",
                "-Command",
                command
            },
            repoRoot,
            "Launch AmandaCore Launcher");

        AppendOutput($"Exit code: {exitCode}");
        SetStatus(exitCode == 0
            ? "Launch AmandaCore Launcher finished with exit code 0."
            : $"Launch AmandaCore Launcher failed with exit code {exitCode}.");
    }

    private Task OpenLogsFolderAsync()
    {
        var candidates = new[]
        {
            Path.Combine(devScriptsRoot, "logs"),
            Path.Combine(repoRoot, "user", "log"),
            Path.Combine(devScriptsRoot, "load-tests"),
            Path.Combine(
                Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
                "amandacore",
                "diagnostics"),
            devScriptsRoot
        };

        var folder = candidates.FirstOrDefault(Directory.Exists);
        if (folder is null)
        {
            AppendOutput("No log or output folder is available.");
            SetStatus("No log or output folder is available.");
            return Task.CompletedTask;
        }

        Process.Start(new ProcessStartInfo
        {
            FileName = "explorer.exe",
            Arguments = "\"" + folder + "\"",
            UseShellExecute = true
        });

        AppendOutput($"Opened folder: {folder}");
        AppendOutput("Exit code: 0");
        SetStatus("Opened logs/output folder.");
        return Task.CompletedTask;
    }

    private Task OpenFolderAsync(string displayName, string folderPath)
    {
        if (!Directory.Exists(folderPath))
        {
            AppendOutput($"{displayName} skipped. Folder not found: {folderPath}");
            SetStatus($"{displayName} skipped.");
            return Task.CompletedTask;
        }

        Process.Start(new ProcessStartInfo
        {
            FileName = "explorer.exe",
            Arguments = "\"" + folderPath + "\"",
            UseShellExecute = true
        });

        AppendOutput($"Opened folder: {folderPath}");
        AppendOutput("Exit code: 0");
        SetStatus($"{displayName} finished with exit code 0.");
        return Task.CompletedTask;
    }

    private Task ResetTestStateAsync()
    {
        var result = MessageBox.Show(
            this,
            "This backs up and resets local account and character test state. Stop the local stack before continuing.",
            "Reset Local Test State",
            MessageBoxButtons.YesNo,
            MessageBoxIcon.Warning);

        if (result != DialogResult.Yes)
        {
            AppendOutput("Reset test state canceled.");
            SetStatus("Reset test state canceled.");
            return Task.CompletedTask;
        }

        return RunPowerShellScriptAsync(
                "Reset test state",
                Path.Combine(repoRoot, "Infra", "qa", "Reset-LocalTestState.ps1"),
                "-All",
                "-ConfirmReset");
    }

    private Task OpenAdminPortalAsync()
    {
        var adminPortalPath = Path.Combine(repoRoot, "Client", "Portal", "admin-portal.html");
        if (!File.Exists(adminPortalPath))
        {
            AppendOutput($"Open admin portal skipped. File not found: {adminPortalPath}");
            SetStatus("Open admin portal skipped.");
            return Task.CompletedTask;
        }

        Process.Start(new ProcessStartInfo
        {
            FileName = adminPortalPath,
            UseShellExecute = true
        });

        AppendOutput($"Opened admin portal: {adminPortalPath}");
        AppendOutput("Exit code: 0");
        SetStatus("Open admin portal finished with exit code 0.");
        return Task.CompletedTask;
    }

    private async Task<int> RunProcessAsync(
        string fileName,
        IReadOnlyCollection<string> arguments,
        string workingDirectory,
        string displayName)
    {
        AppendOutput("");
        AppendOutput($"[{DateTime.Now:HH:mm:ss}] {displayName}");
        AppendOutput($"{fileName} {FormatArguments(arguments)}");

        using var process = new Process();
        process.StartInfo.FileName = fileName;
        process.StartInfo.WorkingDirectory = workingDirectory;
        process.StartInfo.UseShellExecute = false;
        process.StartInfo.RedirectStandardOutput = true;
        process.StartInfo.RedirectStandardError = true;
        process.StartInfo.CreateNoWindow = true;
        foreach (var argument in arguments)
        {
            process.StartInfo.ArgumentList.Add(argument);
        }

        process.OutputDataReceived += (_, args) =>
        {
            if (args.Data is not null)
            {
                AppendOutput(args.Data);
            }
        };
        process.ErrorDataReceived += (_, args) =>
        {
            if (args.Data is not null)
            {
                AppendOutput("[stderr] " + args.Data);
            }
        };

        if (!process.Start())
        {
            throw new InvalidOperationException($"Failed to start {fileName}.");
        }

        process.BeginOutputReadLine();
        process.BeginErrorReadLine();
        await process.WaitForExitAsync();
        return process.ExitCode;
    }

    private void SetBusy(bool busy, string? message = null)
    {
        foreach (var button in commandButtons)
        {
            button.Enabled = !busy;
        }

        UseWaitCursor = busy;
        if (message is not null)
        {
            SetStatus(message);
        }
    }

    private void SetStatus(string message)
    {
        if (InvokeRequired)
        {
            BeginInvoke(() => SetStatus(message));
            return;
        }

        statusLabel.Text = message;
    }

    private void AppendOutput(string message)
    {
        if (InvokeRequired)
        {
            BeginInvoke(() => AppendOutput(message));
            return;
        }

        outputBox.AppendText(message + Environment.NewLine);
    }

    private static string ResolveRepoRoot()
    {
        var startPaths = new[]
        {
            Environment.GetEnvironmentVariable("AMANDACORE_REPO_ROOT"),
            Environment.CurrentDirectory,
            AppContext.BaseDirectory
        };

        foreach (var startPath in startPaths.Where(path => !string.IsNullOrWhiteSpace(path)))
        {
            var directory = new DirectoryInfo(startPath!);
            while (directory is not null)
            {
                if (LooksLikeRepoRoot(directory.FullName))
                {
                    return directory.FullName;
                }

                directory = directory.Parent;
            }
        }

        throw new InvalidOperationException(
            "Unable to resolve the AmandaCore repo root. Start the app from inside the repo or set AMANDACORE_REPO_ROOT.");
    }

    private static bool LooksLikeRepoRoot(string path)
    {
        return File.Exists(Path.Combine(path, "project.json")) &&
            File.Exists(Path.Combine(path, "Infra", "dev", "start-local.ps1")) &&
            Directory.Exists(Path.Combine(path, "Client", "Launcher"));
    }

    private static string ToPowerShellString(string value)
    {
        return "'" + value.Replace("'", "''") + "'";
    }

    private static string FormatArguments(IEnumerable<string> arguments)
    {
        var builder = new StringBuilder();
        foreach (var argument in arguments)
        {
            if (builder.Length > 0)
            {
                builder.Append(' ');
            }

            builder.Append(argument.Contains(' ') ? '"' + argument + '"' : argument);
        }

        return builder.ToString();
    }
}
