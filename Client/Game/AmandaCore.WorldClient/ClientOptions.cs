internal sealed record ClientOptions(string JoinTicketId, string WorldEndpoint, bool AutoDemo)
{
    public static ClientOptions Parse(string[] args)
    {
        string? joinTicket = null;
        var worldEndpoint = "http://localhost:8085";
        var autoDemo = false;

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
            }
        }

        if (string.IsNullOrWhiteSpace(joinTicket))
        {
            throw new InvalidOperationException("A --join-ticket value is required.");
        }

        return new ClientOptions(joinTicket, worldEndpoint, autoDemo);
    }

    private static string GetValue(string[] args, int index, string name)
    {
        if (index >= args.Length)
        {
            throw new InvalidOperationException($"Missing value for {name}.");
        }

        return args[index];
    }
}
