import tensorflow as tf
import tensorflow.keras as keras
from tensorflow.python.keras.engine import data_adapter
from tensorflow.python.eager import backprop


class MNIST(keras.Model):
    def __init__(self):
        super(MNIST, self).__init__()
        self.mnist_model = keras.Sequential([
            tf.keras.Input((28, 28, 1,)),
            tf.keras.layers.Conv2D(32, [3, 3], activation='relu'),
            tf.keras.layers.Conv2D(64, [3, 3], activation='relu'),
            tf.keras.layers.MaxPooling2D(pool_size=(2, 2)),
            tf.keras.layers.Dropout(0.25),
            tf.keras.layers.Flatten(),
            tf.keras.layers.Dense(128, activation='relu'),
            tf.keras.layers.Dropout(0.5),
            tf.keras.layers.Dense(10, activation='softmax')
        ])

    def call(self, inputs, training=None, mask=None):
        pass

    def compile(self, my_opt, loss):
        super(MNIST, self).compile(metrics=['accuracy'])
        self.my_opt = my_opt
        self.loss_fn = loss

    def train_step(self, data):
        data = data_adapter.expand_1d(data)
        x, y, sample_weight = data_adapter.unpack_x_y_sample_weight(data)

        with backprop.GradientTape() as tape:
            y_pred = self.mnist_model(x, training=True)
            loss = self.loss_fn(y, y_pred, sample_weight)

        grads = tape.gradient(loss, self.trainable_variables)
        self.my_opt.apply_gradients(
            zip(grads, self.trainable_variables)
        )
        self.compiled_metrics.update_state(y, y_pred, sample_weight)
        metrics = {m.name: m.result() for m in self.metrics}
        metrics["loss"] = loss
        return metrics

    def test_step(self, data):
        data = data_adapter.expand_1d(data)
        x, y, sample_weight = data_adapter.unpack_x_y_sample_weight(data)

        y_pred = self.mnist_model(x, training=False)
        self.loss_fn(y, y_pred, sample_weight)

        self.compiled_metrics.update_state(y, y_pred, sample_weight)
        return {m.name: m.result() for m in self.metrics}
