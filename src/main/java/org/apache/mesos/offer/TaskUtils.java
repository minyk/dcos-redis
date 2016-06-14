package org.apache.mesos.offer;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import com.google.protobuf.ByteString;
import org.apache.mesos.ExecutorDriver;
import org.apache.mesos.Protos;
import org.apache.mesos.Protos.Label;
import org.apache.mesos.Protos.Labels;
import org.apache.mesos.Protos.TaskID;
import org.apache.mesos.Protos.TaskInfo;
import org.apache.mesos.Protos.TaskStatus;

/**
 * Various utility methods for manipulating data in {@link TaskInfo}s.
 */
public class TaskUtils {
  private static final String TASK_NAME_DELIM = "__";

  /**
   * Converts the unique {@link TaskID} into a Framework defined task name.
   *
   * For example: "instance-0_aoeu5678" => "instance-0"
   */
  public static String toTaskName(TaskID taskId) throws TaskException {
    int underScoreIndex = taskId.getValue().lastIndexOf(TASK_NAME_DELIM);

    if (underScoreIndex == -1) {
      throw new TaskException(
              String.format(
                      "TaskID '%s' is malformed.  Expected '%s' to extract TaskName from TaskID",
                      taskId, TASK_NAME_DELIM));
    }

    return taskId.getValue().substring(0, underScoreIndex);
  }

  /**
   * Converts the Framework defined task name into a unique {@link TaskID}.
   *
   * For example: "instance-0" => "instance-0_aoeu5678"
   */
  public static TaskID toTaskId(String taskName) {
    return TaskID.newBuilder()
        .setValue(taskName + TASK_NAME_DELIM + UUID.randomUUID())
        .build();
  }

  /**
   * Returns whether the provided {@link TaskStatus} shows that the task is in a terminated state.
   */
  public static boolean isTerminated(TaskStatus taskStatus) {
    switch (taskStatus.getState()) {
    case TASK_FINISHED:
    case TASK_FAILED:
    case TASK_KILLED:
    case TASK_LOST:
    case TASK_ERROR:
      return true;
    case TASK_KILLING:
    case TASK_RUNNING:
    case TASK_STAGING:
    case TASK_STARTING:
      break;
    }
    return false;
  }

  /**
   * Ensures that the provided {@link TaskInfo} contains a {@link Label} identifying it as a
   * transient task.
   */
  public static TaskInfo setTransient(TaskInfo taskInfo) {
    Labels labels = setTransient(taskInfo.getLabels());

    return TaskInfo.newBuilder(taskInfo)
      .clearLabels()
      .setLabels(labels)
      .build();
  }

  /**
   * Ensures that the provided {@link TaskInfo} does not contain a {@link Label} identifying it as
   * a transient task.
   */
  public static TaskInfo clearTransient(TaskInfo taskInfo) {
    TaskInfo.Builder taskBuilder = TaskInfo.newBuilder(taskInfo);
    Labels clearedLabels = clearTransient(taskBuilder.getLabels());
    taskBuilder.clearLabels();
    taskBuilder.setLabels(clearedLabels);
    return taskBuilder.build();
  }

  private static Labels clearTransient(Labels labels) {
    Labels.Builder labelBuilder = Labels.newBuilder();

    for (Label label : labels.getLabelsList()) {
      if (!label.getKey().equals(MesosTask.TRANSIENT_FLAG_KEY)) {
        labelBuilder.addLabels(label);
      }
    }

    return labelBuilder.build();
  }

  private static Labels setTransient(Labels labels) {
    labels = clearTransient(labels);

    Labels.Builder labelBuilder = Labels.newBuilder(labels);
    labelBuilder.addLabelsBuilder()
      .setKey(MesosTask.TRANSIENT_FLAG_KEY)
      .setValue("true")
      .build();

    return labelBuilder.build();
  }

  public static Map<String, String> fromEnvironmentToMap(Protos.Environment environment) {
    Map<String, String> map = new HashMap<>();

    final List<Protos.Environment.Variable> variables = environment.getVariablesList();

    for (Protos.Environment.Variable variable : variables) {
      map.put(variable.getName(), variable.getValue());
    }

    return map;
  }

  public static void sendStatus(ExecutorDriver driver,
                                Protos.TaskState state,
                                Protos.TaskID taskID,
                                Protos.SlaveID slaveID,
                                Protos.ExecutorID executorID,
                                String message) {
    sendStatus(driver, state, taskID, slaveID, executorID, message, null);
  }

  public static void sendStatus(ExecutorDriver driver,
                                Protos.TaskState state,
                                Protos.TaskID taskID,
                                Protos.SlaveID slaveID,
                                Protos.ExecutorID executorID,
                                String message,
                                byte[] data) {
    final Protos.TaskStatus.Builder builder = Protos.TaskStatus.newBuilder();

    builder.setState(state);
    builder.setMessage(message);
    builder.setTaskId(taskID);
    builder.setSlaveId(slaveID);
    builder.setExecutorId(executorID);
    builder.setSource(Protos.TaskStatus.Source.SOURCE_EXECUTOR);

    if (data != null) {
      builder.setData(ByteString.copyFrom(data));
    }

    final Protos.TaskStatus taskStatus = builder.build();
    driver.sendStatusUpdate(taskStatus);
  }
}
