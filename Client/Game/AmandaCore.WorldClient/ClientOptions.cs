internal sealed record ClientOptions(
    string JoinTicketId,
    string WorldEndpoint,
    bool AutoDemo,
    StreamingSinkMode StreamingSinkMode,
    string? StreamingCommandFilePath)
{
    public static ClientOptions Parse(string[] args)
    {
        string? joinTicket = null;
        var worldEndpoint = "http://localhost:8085";
        var autoDemo = false;
        var streamingSinkMode = StreamingSinkMode.Console;
        string? streamingCommandFilePath = null;

        for (var index = 0; index < args.Length; index++)
        {
            switch (args[index])
            {
                case "--join-ticket":
                    joinTicket = GetValue(args, ++index, "--join-ticket");
                    break;
                case "--world-endpoint":
                    worldEndpoint = GetValue(args, ++index, "--world-endpoint");
                    break;
                case "--auto-demo":
                    autoDemo = true;
                    break;
                case "--streaming-sink":
                    streamingSinkMode = ParseStreamingSinkMode(GetValue(args, ++index, "--streaming-sink"));
                    break;
                case "--streaming-command-file":
                    streamingCommandFilePath = GetValue(args, ++index, "--streaming-command-file");
                    break;
            }
        }

        if (string.IsNullOrWhiteSpace(joinTicket))
        {
            throw new InvalidOperationException("A --join-ticket value is required.");
        }

        return new ClientOptions(joinTicket, worldEndpoint, autoDemo, streamingSinkMode, streamingCommandFilePath);
    }

    private static string GetValue(string[] args, int index, string name)
    {
        if (index >= args.Length)
        {
            throw new InvalidOperationException($"Missing value for {name}.");
        }

        return args[index];
    }

    private static StreamingSinkMode ParseStreamingSinkMode(string value)
    {
        return value.Trim().ToLowerInvariant() switch
        {
            "console" => StreamingSinkMode.Console,
            "scene-commands" => StreamingSinkMode.SceneCommands,
            "both" => StreamingSinkMode.Both,
            _ => throw new InvalidOperationException("Invalid --streaming-sink value. Expected console, scene-commands, or both.")
        };
    }
}

internal enum StreamingSinkMode
{
    Console,
    SceneCommands,
    Both
}
