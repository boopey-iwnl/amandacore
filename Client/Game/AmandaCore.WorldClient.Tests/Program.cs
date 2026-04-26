var tests = new StreamingPreviewAdapterTests();
tests.InitialFrameEmitsStablePlaceholderCommands();
tests.CellChangesEmitHideAndHighlightCommands();
tests.DisabledStreamingClearsPlaceholderState();
tests.StreamingSinkModeParsesSceneCommands();
tests.StreamingCommandFileWritesDeterministicJsonLines();
tests.StreamingCommandFileAllowsLiveBridgeReader();
Console.WriteLine("AmandaCore.WorldClient streaming adapter tests passed.");

internal sealed class StreamingPreviewAdapterTests
{
    public void InitialFrameEmitsStablePlaceholderCommands()
    {
        var recorder = new RecordingPlaceholderSceneCommandSink();
        var preview = new ClientStreamingPreviewState(new PlaceholderSceneStreamingAdapter(recorder));

        preview.Update(TestWorldSessionResponses.Create(zoneId: "dawnwake_landing", playerX: 5, playerY: 5));

        AssertCommands(
            recorder.Commands,
            PlaceholderSceneCommandNames.CreateZoneBoundsVolume,
            PlaceholderSceneCommandNames.CreateStreamingCellVolume,
            PlaceholderSceneCommandNames.CreateStreamingCellVolume,
            PlaceholderSceneCommandNames.HighlightCurrentCell,
            PlaceholderSceneCommandNames.ShowTransitionAffordance);
        AssertEqual("dawnwake_landing", recorder.Commands[0].ZoneId, "zone bounds zone id");
        AssertEqual("dw_map_landing", recorder.Commands[0].MapId, "zone bounds map id");
        AssertEqual("landing_west", recorder.Commands[3].CellId, "current cell");
        AssertEqual(false, recorder.Commands[4].Ready, "transition ready state");
    }

    public void CellChangesEmitHideAndHighlightCommands()
    {
        var recorder = new RecordingPlaceholderSceneCommandSink();
        var preview = new ClientStreamingPreviewState(new PlaceholderSceneStreamingAdapter(recorder));
        preview.Update(TestWorldSessionResponses.Create(zoneId: "dawnwake_landing", playerX: 5, playerY: 5));
        recorder.Clear();

        preview.Update(TestWorldSessionResponses.Create(zoneId: "dawnwake_landing", playerX: 75, playerY: 5, includeEastCell: true));

        AssertCommands(
            recorder.Commands,
            PlaceholderSceneCommandNames.CreateStreamingCellVolume,
            PlaceholderSceneCommandNames.HighlightCurrentCell,
            PlaceholderSceneCommandNames.ShowTransitionAffordance);
        AssertEqual("landing_east", recorder.Commands[0].CellId, "new visible cell");
        AssertEqual("landing_east", recorder.Commands[1].CellId, "highlighted cell");
    }

    public void DisabledStreamingClearsPlaceholderState()
    {
        var recorder = new RecordingPlaceholderSceneCommandSink();
        var preview = new ClientStreamingPreviewState(new PlaceholderSceneStreamingAdapter(recorder));

        preview.Update(TestWorldSessionResponses.Create(zoneId: "dawnwake_landing", playerX: 5, playerY: 5));
        recorder.Clear();

        preview.Update(new WorldSessionResponse
        {
            ZoneId = "dawnwake_landing",
            Position = new Position { X = 5, Y = 5, Z = 0 },
            Streaming = new StreamingStateResponse { Enabled = false, ZoneId = "dawnwake_landing" }
        });

        AssertCommands(
            recorder.Commands,
            PlaceholderSceneCommandNames.HideStreamingCellVolume,
            PlaceholderSceneCommandNames.HideStreamingCellVolume,
            PlaceholderSceneCommandNames.ClearCurrentCellHighlight,
            PlaceholderSceneCommandNames.ClearTransitionAffordance);
    }

    public void StreamingSinkModeParsesSceneCommands()
    {
        var options = ClientOptions.Parse([
            "--join-ticket",
            "ticket",
            "--streaming-sink",
            "scene-commands",
            "--streaming-command-file",
            "streaming.commands.jsonl"
        ]);

        AssertEqual(StreamingSinkMode.SceneCommands, options.StreamingSinkMode, "streaming sink mode");
        AssertEqual("streaming.commands.jsonl", options.StreamingCommandFilePath, "streaming command file");
    }

    public void StreamingCommandFileWritesDeterministicJsonLines()
    {
        var commandPath = Path.Combine(Path.GetTempPath(), $"amandacore-streaming-{Guid.NewGuid():N}.jsonl");
        try
        {
            var preview = new ClientStreamingPreviewState(new PlaceholderSceneStreamingAdapter(new JsonLinesPlaceholderSceneCommandSink(commandPath)));

            preview.Update(TestWorldSessionResponses.Create(zoneId: "dawnwake_landing", playerX: 5, playerY: 5));

            var lines = File.ReadAllLines(commandPath);
            AssertEqual(5, lines.Length, "command line count");
            if (!lines[0].Contains("\"Command\":\"CreateZoneBoundsVolume\"", StringComparison.Ordinal))
            {
                throw new InvalidOperationException($"Expected first command line to create zone bounds, got {lines[0]}.");
            }
            if (!lines[3].Contains("\"Command\":\"HighlightCurrentCell\"", StringComparison.Ordinal))
            {
                throw new InvalidOperationException($"Expected fourth command line to highlight current cell, got {lines[3]}.");
            }
        }
        finally
        {
            if (File.Exists(commandPath))
            {
                File.Delete(commandPath);
            }
        }
    }

    public void StreamingCommandFileAllowsLiveBridgeReader()
    {
        var commandPath = Path.Combine(Path.GetTempPath(), $"amandacore-streaming-live-{Guid.NewGuid():N}.jsonl");
        try
        {
            var preview = new ClientStreamingPreviewState(new PlaceholderSceneStreamingAdapter(new JsonLinesPlaceholderSceneCommandSink(commandPath)));
            using var reader = new FileStream(commandPath, FileMode.Open, FileAccess.Read, FileShare.ReadWrite);

            preview.Update(TestWorldSessionResponses.Create(zoneId: "dawnwake_landing", playerX: 5, playerY: 5));

            reader.Seek(0, SeekOrigin.Begin);
            using var streamReader = new StreamReader(reader, leaveOpen: true);
            var commandStream = streamReader.ReadToEnd();
            if (!commandStream.Contains("\"Command\":\"CreateStreamingCellVolume\"", StringComparison.Ordinal))
            {
                throw new InvalidOperationException("Expected live bridge reader to observe streaming cell commands while the writer is active.");
            }
        }
        finally
        {
            if (File.Exists(commandPath))
            {
                File.Delete(commandPath);
            }
        }
    }

    private static void AssertCommands(IReadOnlyList<PlaceholderSceneCommand> commands, params string[] expected)
    {
        AssertEqual(expected.Length, commands.Count, "command count");
        for (var index = 0; index < expected.Length; index++)
        {
            AssertEqual(expected[index], commands[index].Command, $"command {index}");
        }
    }

    private static void AssertEqual<T>(T expected, T actual, string label)
    {
        if (!EqualityComparer<T>.Default.Equals(expected, actual))
        {
            throw new InvalidOperationException($"Expected {label} to be {expected}, got {actual}.");
        }
    }
}

internal static class TestWorldSessionResponses
{
    public static WorldSessionResponse Create(string zoneId, double playerX, double playerY, bool includeEastCell = false)
    {
        var cells = new List<StreamingCellResponse>
        {
            new()
            {
                CellId = "landing_west",
                DisplayName = "Landing West",
                Priority = 20,
                Bounds = new MapBoundsResponse { MinX = 0, MinY = 0, MinZ = -1, MaxX = 50, MaxY = 50, MaxZ = 10 }
            },
            new()
            {
                CellId = "landing_transition",
                DisplayName = "Landing Transition",
                Priority = 10,
                Bounds = new MapBoundsResponse { MinX = 0, MinY = 0, MinZ = -1, MaxX = 100, MaxY = 100, MaxZ = 10 }
            }
        };

        if (includeEastCell)
        {
            cells.Add(new StreamingCellResponse
            {
                CellId = "landing_east",
                DisplayName = "Landing East",
                Priority = 30,
                Bounds = new MapBoundsResponse { MinX = 60, MinY = 0, MinZ = -1, MaxX = 100, MaxY = 50, MaxZ = 10 }
            });
        }

        return new WorldSessionResponse
        {
            ZoneId = zoneId,
            DisplayName = "Test",
            Position = new Position { X = playerX, Y = playerY, Z = 0 },
            Streaming = new StreamingStateResponse
            {
                Enabled = true,
                ZoneId = zoneId,
                MapId = "dw_map_landing",
                Bounds = new MapBoundsResponse { MinX = 0, MinY = 0, MinZ = -1, MaxX = 100, MaxY = 100, MaxZ = 10 },
                AdjacentZoneIds = ["dawnwake_tideglass_shoal"],
                StreamingCells = cells,
                TransitionHints =
                [
                    new()
                    {
                        TransitionId = "to_tideglass",
                        DisplayName = "To Tideglass",
                        TargetZoneId = "dawnwake_tideglass_shoal",
                        DestinationEntryId = "from_landing",
                        StreamingCellId = "landing_transition",
                        Hint = "Causeway",
                        Position = new Position { X = 90, Y = 5, Z = 0 },
                        Radius = 5
                    }
                ]
            }
        };
    }
}
