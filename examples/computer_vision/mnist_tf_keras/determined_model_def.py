import tensorflow as tf

from determined.keras import TFKerasTrial

from mnist import MNIST


class MnistTrial(TFKerasTrial):
    def __init__(self, context):
        self.context = context

    def build_model(self):
        model = self.context.wrap_model(MNIST())
        optimizer = self.context.wrap_optimizer(tf.keras.optimizers.Adam(0.001))
        model.compile(optimizer, loss=tf.losses.SparseCategoricalCrossentropy())

        return model

    def build_training_data_loader(self):
        (mnist_images, mnist_labels), _ = \
            tf.keras.datasets.mnist.load_data(path='mnist-%d.npz' % self.context.distributed.get_rank())

        dataset = tf.data.Dataset.from_tensor_slices(
            (tf.cast(mnist_images[..., tf.newaxis] / 255.0, tf.float32),
             tf.cast(mnist_labels, tf.int64))
        )
        dataset = self.context.wrap_dataset(dataset)
        dataset = dataset.repeat().shuffle(10000).batch(self.context.get_per_slot_batch_size())
        return dataset

    def build_validation_data_loader(self):
        (mnist_images, mnist_labels), _ = \
            tf.keras.datasets.mnist.load_data(path='mnist-%d.npz' % self.context.distributed.get_rank())

        dataset = tf.data.Dataset.from_tensor_slices(
            (tf.cast(mnist_images[..., tf.newaxis] / 255.0, tf.float32),
             tf.cast(mnist_labels, tf.int64))
        )
        dataset = self.context.wrap_dataset(dataset)
        dataset = dataset.shuffle(10000).batch(self.context.get_per_slot_batch_size())
        return dataset
