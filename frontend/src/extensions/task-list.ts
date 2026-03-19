import TaskList from "@tiptap/extension-task-list";
import TaskItem from "@tiptap/extension-task-item";

const ConfiguredTaskList = TaskList;
const ConfiguredTaskItem = TaskItem.configure({ nested: true });

export { ConfiguredTaskList as TaskList, ConfiguredTaskItem as TaskItem };
