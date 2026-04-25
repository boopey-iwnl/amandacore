using System.Text.Json;

namespace AmandaCore.Launcher.Models;

internal sealed class LauncherSession
{
    public string AccessToken { get; set; } = string.Empty;
    public string RefreshToken { get; set; } = string.Empty;
    public string AccountId { get; set; } = string.Empty;

    public static LauncherSession? Load()
    {
        var path = GetPath();
        if (!File.Exists(path))
        {
            return null;
        }

        var json = File.ReadAllText(path);
        return JsonSerializer.Deserialize<LauncherSession>(json);
    }

    public void Save()
    {
        var path = GetPath();
        Directory.CreateDirectory(Path.GetDirectoryName(path)!);
        File.WriteAllText(path, JsonSerializer.Serialize(this, new JsonSerializerOptions { WriteIndented = true }));
    }

    public static void Clear()
    {
        var path = GetPath();
        if (File.Exists(path))
        {
            File.Delete(path);
        }
    }

    private static string GetPath()
    {
        var root = Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
            "amandacore");
        return Path.Combine(root, "launcher-session.json");
    }
}
