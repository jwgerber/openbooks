import {
  ActionIcon,
  Button,
  Drawer,
  Group,
  Paper,
  Stack,
  Switch,
  Text,
  TextInput,
  Tooltip
} from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { DotsSixVertical, X } from "phosphor-react";
import { DragEvent, FormEvent, useEffect, useState } from "react";
import {
  Settings,
  useGetSettingsQuery,
  useUpdateSettingsMutation
} from "../../state/api";
import { useAppSelector } from "../../state/store";

// A search counts as in flight until results arrive or this much time passes,
// so a lost "no results" reply can't disable Save forever.
const SEARCH_IN_FLIGHT_MS = 60_000;

interface Props {
  opened: boolean;
  onClose: () => void;
}

export default function SettingsDrawer({ opened, onClose }: Props) {
  const { data: saved } = useGetSettingsQuery(null);
  const [updateSettings, { isLoading: isSaving }] = useUpdateSettingsMutation();
  const [form, setForm] = useState<Settings | undefined>(saved);
  const [error, setError] = useState<string>();
  const [newFormat, setNewFormat] = useState("");
  const [dragIndex, setDragIndex] = useState<number>();

  const inFlightDownloads = useAppSelector(
    (store) => store.state.inFlightDownloads.length
  );
  const activeItem = useAppSelector((store) => store.state.activeItem);
  const isSearching =
    !!activeItem &&
    activeItem.results === undefined &&
    activeItem.errors === undefined &&
    Date.now() - activeItem.timestamp < SEARCH_IN_FLIGHT_MS;
  const busy = inFlightDownloads > 0 || isSearching;

  useEffect(() => {
    if (opened) {
      setForm(saved);
      setError(undefined);
    }
  }, [opened, saved]);

  if (!form) {
    return null;
  }

  const set = <K extends keyof Settings>(key: K, value: Settings[K]) =>
    setForm({ ...form, [key]: value });

  const moveFormat = (from: number, to: number) => {
    const order = [...form.formatOrder];
    const [moved] = order.splice(from, 1);
    order.splice(to, 0, moved);
    set("formatOrder", order);
  };

  const addFormat = () => {
    const format = newFormat.trim().toLowerCase().replace(/^\./, "");
    if (format && !form.formatOrder.includes(format)) {
      set("formatOrder", [...form.formatOrder, format]);
    }
    setNewFormat("");
  };

  const save = async (event: FormEvent) => {
    event.preventDefault();
    setError(undefined);
    try {
      await updateSettings(form).unwrap();
      showNotification({ message: "Settings saved — reconnecting" });
      onClose();
    } catch (err: any) {
      setError(err?.data?.error ?? "Unable to save settings.");
    }
  };

  return (
    <Drawer
      opened={opened}
      onClose={onClose}
      title={<Text weight="bold">Settings</Text>}
      padding="md"
      size="md"
      position="right">
      <form onSubmit={save}>
        <Stack spacing="sm">
          <TextInput
            label="Server"
            description="host:port"
            value={form.server}
            onChange={(e) => set("server", e.currentTarget.value)}
          />
          <Switch
            label="Secure connection (TLS)"
            checked={form.enableTls}
            onChange={(e) => set("enableTls", e.currentTarget.checked)}
          />
          <TextInput
            label="Channel"
            icon={<Text size="sm">#</Text>}
            value={form.channel}
            onChange={(e) => set("channel", e.currentTarget.value)}
          />
          <TextInput
            label="Search trigger"
            icon={<Text size="sm">@</Text>}
            value={form.searchBot}
            onChange={(e) => set("searchBot", e.currentTarget.value)}
          />
          <TextInput
            label="Nickname"
            placeholder="random"
            value={form.userName}
            onChange={(e) => set("userName", e.currentTarget.value)}
          />

          <div>
            <Text size="sm" weight={500}>
              Format order
            </Text>
            <Text size="xs" color="dimmed" mb={4}>
              Drag to reorder. Top is preferred.
            </Text>
            <Stack spacing={4}>
              {form.formatOrder.map((format, index) => (
                <Paper
                  key={format}
                  withBorder
                  p={4}
                  draggable
                  onDragStart={() => setDragIndex(index)}
                  onDragOver={(e: DragEvent) => e.preventDefault()}
                  onDrop={() => {
                    if (dragIndex !== undefined && dragIndex !== index) {
                      moveFormat(dragIndex, index);
                    }
                    setDragIndex(undefined);
                  }}
                  style={{ cursor: "grab" }}>
                  <Group position="apart" noWrap>
                    <Group spacing="xs" noWrap>
                      <DotsSixVertical size={16} />
                      <Text size="sm">{format}</Text>
                    </Group>
                    <ActionIcon
                      size="sm"
                      aria-label={`Remove ${format}`}
                      onClick={() =>
                        set(
                          "formatOrder",
                          form.formatOrder.filter((f) => f !== format)
                        )
                      }>
                      <X size={14} />
                    </ActionIcon>
                  </Group>
                </Paper>
              ))}
            </Stack>
            <Group mt={6} spacing="xs" noWrap>
              <TextInput
                size="xs"
                placeholder="Add format, e.g. azw"
                value={newFormat}
                onChange={(e) => setNewFormat(e.currentTarget.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addFormat();
                  }
                }}
                style={{ flexGrow: 1 }}
              />
              <Button size="xs" variant="default" onClick={addFormat}>
                Add
              </Button>
            </Group>
          </div>

          <TextInput
            label="Download folder"
            description="Set by the container's /books volume."
            value={form.downloadDir}
            readOnly
            disabled
          />

          {error && (
            <Text size="sm" color="red">
              {error}
            </Text>
          )}

          <Tooltip
            label="Wait for the current search/download to finish"
            disabled={!busy}>
            <div>
              <Button
                type="submit"
                fullWidth
                loading={isSaving}
                disabled={busy}>
                Save
              </Button>
            </div>
          </Tooltip>
        </Stack>
      </form>
    </Drawer>
  );
}
